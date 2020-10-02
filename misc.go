package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"regexp"

	"github.com/bwmarrin/discordgo"
)

func containsUsr(l []*discordgo.User, k *discordgo.User) bool {
	for _, i := range l {
		if i.ID == k.ID {
			return true
		}
	}
	return false
}

// Registers emoji with g's icon under conf.EmoteGuild, returns registered emoji or nil
func registerGuildEmoji(s *discordgo.Session, g *discordgo.Guild) *discordgo.Emoji {
	// TODO Handle gifs
	// FIXME Don't overload return value as error
	img, err := s.GuildIcon(g.ID)
	if err != nil {
		fmt.Printf("server %s likely has no icon, ignoring (err: %s)\n", g.ID, err)
		return nil
	}
	bd := img.Bounds()
	cn := center(bd)
	circleImg := image.NewRGBA(bd)
	draw.DrawMask(circleImg, bd, img, image.ZP, &circle{cn, bd.Min.X - cn.X}, image.ZP, draw.Over)
	iUrl, err := encodeImg(circleImg)
	if err != nil {
		fmt.Printf("failed to encode icon for %s: %s\n", g.ID, err)
		return nil
	}
	e, err := s.GuildEmojiCreate(conf.EmoteGuild, g.ID, iUrl, nil)
	if err != nil {
		fmt.Printf("failed to add emoji for guild %s: %s\n", g.ID, err)
		return nil
	}
	return e
}

// Returns an emoji for the corresponding GOOS value, or ❓ if we don't have one
// FIXME Current values are specific to our instance
func osEmoji(s string) string {
	emojis := map[string]string{
		"windows": "<:windwos:758861126271631390>",
		"linux":   "<:tux:758874037706948648>",
		"solaris": "<:solaris:758875213961232404>",
		"openbsd": "<:puffy:758875557235654657>",
		"netbsd":  "<:netbsd:758875679961514014>",
		//		"plan9":     "<:spaceglenda:758857214596874241>",
		"plan9":     "<:glenda:758886314438295553>",
		"freebsd":   "<:freebased:758864143792078910>",
		"dragonfly": "<:dragonfly:758865198941077535>",
		"darwin":    "<:applel:758863829764931625>"}

	if emojis[s] == "" {
		return "❓"
	}
	return emojis[s]
}

// Generates comma-separated list of user mentions
func mentions(usrs []string) (m string) {
	for _, u := range usrs {
		m = fmt.Sprintf("%s, <@%s>", m, u)
	}
	m = m[2:]
	return
}

// List member IDs currently connected under voice channel with id cid on guild gid
func getCallMembers(s *discordgo.Session, cid, gid string) (usrs []string) {
	g, err := s.State.Guild(gid)
	if err != nil {
		fmt.Println("getCallMembers: cannot get guild state:", err)
		return nil
	}
	for _, vs := range g.VoiceStates {
		if vs.ChannelID == cid {
			usrs = append(usrs, vs.UserID)
		}
	}
	return
}

// Gets current VoiceState for a given uid under guild gid
func getVoiceState(s *discordgo.Session, uid, gid string) *discordgo.VoiceState {
	g, err := s.State.Guild(gid)
	if err != nil {
		fmt.Println("getVoiceChan: cannot get guild state:", err)
		return nil
	}
	for _, vs := range g.VoiceStates {
		if vs.UserID == uid {
			return vs
		}
	}
	return nil
}

// Verifies if a string represents a possibly valid Among Us match code
func maybeValidCode(c string) bool {
	r, err := regexp.Match("^[A-Za-z]{6}$", []byte(c))
	if err != nil {
		fmt.Println("maybeValidCode:", err)
		return false
	}
	return r
}

// Transforms image i into a png data uri
func encodeImg(i image.Image) (string, error) {
	out := new(bytes.Buffer)
	err := png.Encode(out, i)
	if err != nil {
		return "", err
	}
	b64str := base64.StdEncoding.EncodeToString(out.Bytes())
	return fmt.Sprint("data:image/png;base64,", b64str), nil
}

// Returns central point for rectangle r
func center(rect image.Rectangle) image.Point {
	return image.Point{(rect.Max.X - rect.Min.X) / 2, (rect.Max.Y - rect.Min.Y) / 2}
}

/* Shamelessly copy-pasted from https://blog.golang.org/image-draw */

type circle struct {
	p image.Point
	r int
}

func (c *circle) ColorModel() color.Model {
	return color.AlphaModel
}

func (c *circle) Bounds() image.Rectangle {
	return image.Rect(c.p.X-c.r, c.p.Y-c.r, c.p.X+c.r, c.p.Y+c.r)
}

func (c *circle) At(x, y int) color.Color {
	xx, yy, rr := float64(x-c.p.X)+0.5, float64(y-c.p.Y)+0.5, float64(c.r)
	if xx*xx+yy*yy < rr*rr {
		return color.Alpha{255}
	}
	return color.Alpha{0}
}

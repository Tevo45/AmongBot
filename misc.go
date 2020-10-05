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
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// FIXME Stop codebase from degenerating into chaos

func containsUsr(l []*discordgo.User, k *discordgo.User) bool {
	for _, i := range l {
		if i.ID == k.ID {
			return true
		}
	}
	return false
}

var emojiCache = map[string]*discordgo.Emoji{}

// Registers emoji with g's icon under conf.EmoteGuild, returns registered emoji or nil
func registerGuildEmoji(s *discordgo.Session, g *discordgo.Guild) *discordgo.Emoji {
	// TODO Handle gifs
	// FIXME Don't overload return value as error
	if cached := emojiCache[g.ID]; cached != nil {
		return cached
	}
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
	emojiCache[g.ID] = e
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

// Creates a timer of duration t and registers it on *dict with key k, keeping it there until it ticks once
func regTimer(t time.Duration, dict *map[string]<-chan time.Time, k string) <-chan time.Time {
	clock := time.After(t)
	if dict != nil {
		(*dict)[k] = clock
	}
	c := make(chan time.Time, 1)
	go func() {
		tmp := <-clock
		if dict != nil {
			delete(*dict, k)
		}
		c <- tmp
	}()
	return c
}

// Destroys a message after t ticks once
func selfDestruct(s *discordgo.Session, m *discordgo.Message, t <-chan time.Time) {
	<-t
	s.ChannelMessageDelete(m.ChannelID, m.ID)
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

// This could be in a library
// FIXME Error check everything below this

type menuEntry interface {
	Content() string
}

type menuProvider interface {
	Get(idx, count int) []menuEntry
	Len() int
}

type menuSlice struct {
	slice []menuEntry
}

func sliceMenu(s []menuEntry) menuProvider {
	return &menuSlice{slice: s}
}

func (m *menuSlice) Get(idx, c int) []menuEntry {
	if idx > len(m.slice) {
		return []menuEntry{}
	}
	r := m.slice[idx:]
	if c < len(r) {
		r = r[:c]
	}
	return r
}

func (m *menuSlice) Len() int {
	return len(m.slice)
}

type styleFilter interface {
	Filter(em *discordgo.MessageEmbed, curPg int, itemsPerPg int)
}

type styleFun struct {
	fn func(em *discordgo.MessageEmbed, curPg int, itemsPerPg int)
}

func (s *styleFun) Filter(em *discordgo.MessageEmbed, curPg int, itemsPerPg int) {
	s.fn(em, curPg, itemsPerPg)
}

func styleFunc(fn func(em *discordgo.MessageEmbed, curPg int, itemsPerPg int)) styleFilter {
	return &styleFun{fn: fn}
}

var (
	rmRegister = map[string]*reactionMenu{}
	forward    = "rightamong:761818451387351051" // TODO Assets, move this to an external file
	backward   = "leftamong:761818430579539978"
)

func rmReactionHandler(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.UserID == s.State.User.ID {
		return
	}
	menu := rmRegister[r.MessageID]
	if menu == nil {
		return
	}
	switch r.Emoji.APIName() {
	case forward:
		menu.Forward(s)
	case backward:
		menu.Backward(s)
	}
}

func reactionSlider(s *discordgo.Session, ch string, fields menuProvider, filter styleFilter) (err error, rm *reactionMenu) {
	rm = &reactionMenu{fields: fields, itemsPerPg: 10, filter: filter}
	rm.msg, err = s.ChannelMessageSendEmbed(ch, rm.Render())
	if err != nil {
		rm = nil
		return
	}
	rm.UpdateReactions(s)
	rmRegister[rm.msg.ID] = rm
	return
}

type reactionMenu struct {
	fields     menuProvider
	msg        *discordgo.Message
	curPg      int
	itemsPerPg int
	filter     styleFilter
}

func (m *reactionMenu) Forward(s *discordgo.Session) {
	if m.curPg >= m.maxPg() {
		return
	}
	m.curPg++
	m.Update(s)
}

func (m *reactionMenu) Backward(s *discordgo.Session) {
	if m.curPg <= 0 {
		return
	}
	m.curPg--
	m.Update(s)
}

func (m *reactionMenu) Render() *discordgo.MessageEmbed {
	fls := m.fields.Get(m.curPg*m.itemsPerPg, m.itemsPerPg)
	if len(fls) == 0 {
		return &discordgo.MessageEmbed{Title: "Illegal page"}
	}
	pgBody := ""
	wrks := make([]chan string, 0)
	for _, fl := range fls {
		chann := make(chan string, 1)
		wrks = append(wrks, chann)
		go func(c chan string, f menuEntry) {
			c <- f.Content() + "\n"
		}(chann, fl)
	}
	for _, c := range wrks {
		pgBody += <-c
	}
	em := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Página %d", m.curPg+1),
		Description: pgBody,
	}
	if m.filter != nil {
		m.filter.Filter(em, m.curPg, m.itemsPerPg)
	}
	return em
}

func (m *reactionMenu) Update(s *discordgo.Session) {
	m.UpdateEmbed(s)
	m.UpdateReactions(s)
}

func (m *reactionMenu) UpdateEmbed(s *discordgo.Session) {
	s.ChannelMessageEditEmbed(m.msg.ChannelID, m.msg.ID, m.Render())
}

func (m *reactionMenu) UpdateReactions(s *discordgo.Session) {
	s.MessageReactionsRemoveAll(m.msg.ChannelID, m.msg.ID)
	if m.curPg > 0 {
		s.MessageReactionAdd(m.msg.ChannelID, m.msg.ID, backward)
	}
	if m.curPg < m.maxPg() {
		s.MessageReactionAdd(m.msg.ChannelID, m.msg.ID, forward)
	}
}

func (m *reactionMenu) maxPg() (r int) {
	l := m.fields.Len() - 1
	i := m.itemsPerPg
	r = l / i
	return
}

type stubItem struct{}

func (stubItem) Content() string {
	return "stub"
}

type srvProvider struct {
	session *discordgo.Session
}

func guildProvider(s *discordgo.Session) menuProvider {
	return &srvProvider{session: s}
}

func (p *srvProvider) Get(idx, c int) (r []menuEntry) {
	r = []menuEntry{}
	if idx > p.Len() {
		return
	}
	pr := state.GetPremiumGuilds()
	fmt.Println(pr)
	if len(pr) > idx {
		for i := idx; i < len(pr) && c != 0; i++ {
			fmt.Println(i)
			g, err := p.session.Guild(pr[i].ID)
			fmt.Println(g, err)
			if err != nil {
				fmt.Println("Unable to fetch guild:", err)
				continue
			}
			r = append(r, &srvItem{
				session:    p.session,
				guild:      g,
				premium:    true,
				premiumUrl: pr[i].InviteURL,
			})
			c--
		}
	}
	// FIXME Premium guilds show up 2 times in the menu
	gl := p.session.State.Guilds[idx:]
	if len(gl) > c {
		gl = gl[:c]
	}
	for _, g := range gl {
		r = append(r, &srvItem{session: p.session, guild: g})
	}
	return
}

func (p *srvProvider) Len() int {
	return len(p.session.State.Guilds)
}

type srvItem struct {
	session    *discordgo.Session
	guild      *discordgo.Guild
	premium    bool
	premiumUrl string
}

func (i *srvItem) Content() (s string) {
	/*
		e := registerGuildEmoji(i.session, i.guild)
		if e != nil {
			go func() {
				// FIXME Better cleanup system
				// We don't know when it's safe to actually remove, so we just assume this is enough
				<-time.After(15 * time.Second)
				delete(emojiCache, i.guild.ID)
				err := i.session.GuildEmojiDelete(conf.EmoteGuild, e.ID)
				if err != nil {
					fmt.Printf("failed to remove listing emoji for guild %s: %s\n", i.guild.ID, err)
				}
			}()
			s += e.MessageFormat()
		} else {
			s += "<:default:761420942072610817>" // TODO Move this to a resource file
		}
	*/
	s += "<:default:761420942072610817>"
	if i.premium {
		s += fmt.Sprintf("• [%s](%s) - <:pr:761394323778437121><:em:761393700177575978><:iu:761393724365209620><:m_:761393746385829938>", i.guild.Name, i.premiumUrl) // TODO Same as above
	} else {
		s += fmt.Sprintf("∙ %s", i.guild.Name)
	}
	return
}

func tagToMap(tag string) map[string]*string {
	m := map[string]*string{}
	fields := strings.Split(tag, " ")
	for _, field := range fields {
		pair := strings.Split(field, ":")
		if len(pair) == 0 {
			continue // What the fuck
		}
		if len(pair) == 1 {
			m[pair[0]] = nil
			continue
		}
		m[pair[0]] = &pair[1] // Maybe we should concatenate pair[1:] instead?
	}
	return m
}

// We could make this variadic, but so far I haven't felt the need to
func coalesce(e1, e2 error) error {
	if e1 == nil {
		return e2
	}
	if e2 == nil {
		return e1
	}
	return fmt.Errorf("%w; %w", e1, e2)
}

func hasPermissions(s *discordgo.Session, m *discordgo.Member, perms int) bool {
	for _, rId := range m.Roles {
		role, err := s.State.Role(m.GuildID, rId)
		if err != nil {
			fmt.Printf("Unable to fetch role %s: %s\n", rId, err)
			continue
		}
		if role.Permissions & perms != 0 {
			return true
		}
	}
	return false
}

//  Copy-pasted from https://github.com/bwmarrin/discordgo/wiki/FAQ#determining-if-a-role-has-a-permission
func MemberHasPermission(s *discordgo.Session, guildID string, userID string, permission int) (bool, error) {
	member, err := s.State.Member(guildID, userID)
	if err != nil {
		if member, err = s.GuildMember(guildID, userID); err != nil {
			return false, err
		}
	}

    // Iterate through the role IDs stored in member.Roles
    // to check permissions
	for _, roleID := range member.Roles {
		role, err := s.State.Role(guildID, roleID)
		if err != nil {
			return false, err
		}
		if role.Permissions&permission != 0 {
			return true, nil
		}
	}

	return false, nil
}

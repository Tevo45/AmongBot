
TARGETDIR=/cfg/$sysname/autostart
GOSRC=`{walk | grep '\.go$'}

amongbot: $GOSRC
	go build

run: amongbot
	amongbot

install promote:V: $TARGETDIR/amongbot

$TARGETDIR/%: %
	cp $prereq $target 

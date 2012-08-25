package main

import (
	"bufio"
	"flag"
	"fmt"
	irc "github.com/fluffle/goirc/client"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var usedMeMap map[string]time.Time
var currentUrls map[string]bool
var levelTwoUsername string
var levelTwoServer string

func lockUsedMe(hostname string) bool {
	currentDate, ok := usedMeMap[hostname]
	if !ok {
		usedMeMap[hostname] = time.Now()
	} else {
		// now we need to look it up
		if time.Since(currentDate) > (10 * time.Minute) {
			// we'll let them off
			usedMeMap[hostname] = time.Now()
		} else {
			// nope
			return false
		}
	}
	return true
}

func lockUrl(url string) bool {
	_, ok := currentUrls[url]
	if !ok {
		currentUrls[url] = true
		return true
	}
	return false
}

func unlockUrl(url string) {
	delete(currentUrls, url)
}

func sendPublicMessage(conn *irc.Conn, sendingTo string, replyingTo string, serverId string, userId string, message string) {
	conn.Privmsg(sendingTo, fmt.Sprintf("%s: (08-%s - %s) %s", replyingTo, serverId, userId, message))
}

func doTheBreak(conn *irc.Conn, urlToUse string, theirNick string, replyTo string, serverId string, userId string) {
	defer unlockUrl(urlToUse)

	// we need to open an SSH connection to our host
	// we'll just use exec for this, because it knows about our keys
	sshCommand := exec.Command("ssh", levelTwoUsername+"@"+levelTwoServer, "./pingit_recvit", "-servermode", urlToUse, levelTwoServer)
	// grab stdout
	stdoutPipe, err := sshCommand.StdoutPipe()
	stdout := bufio.NewReader(stdoutPipe)
	if err != nil {
		log.Printf("Hit a snag whilst getting the stdoutpipe: %s\n", err.Error())
		conn.Privmsg(replyTo, theirNick+": Oops - something went wrong finding the flag. Maybe some other time.")
		return
	}

	err = sshCommand.Start()
	if err != nil {
		log.Printf("Hit a snag whilst starting the command: %s\n", err.Error())
		conn.Privmsg(replyTo, theirNick+": Oops - something went wrong finding the flag. Maybe some other time.")
		return
	}

	// now we just keep reading
	var startedTime time.Time
	for {
		lineComplete := false
		lineStr := ""
		for !lineComplete {
			lineBytes, isPrefix, err := stdout.ReadLine()
			if err == io.EOF {
				return
			}
			lineStr += string(lineBytes)
			lineComplete = !isPrefix
		}

		log.Printf("%s: REPORTED %s\n", urlToUse, lineStr)
		if strings.HasPrefix(lineStr, "STARTED") {
			startedTime = time.Now()
		} else if strings.HasPrefix(lineStr, "BROKE ") {
			// now we tell them, but not much :)
			lineBits := strings.Split(lineStr, " ")
			chunkBrokenInt64, _ := strconv.ParseInt(lineBits[1], 10, 0)
			//actualNumberInt64, err := strconv.ParseInt(lineBits[2], 10, 0)
			// tell them
			sendPublicMessage(conn, replyTo, theirNick, serverId, userId, fmt.Sprintf("I've broken %d%% of your flag!", (chunkBrokenInt64+1)*25))
		} else if strings.HasPrefix(lineStr, "BROKE_ALL ") {
			// now we tell them, but not much :)
			//lineBits := strings.Split(lineStr, " ")
			//chunkBrokenInt64, err := strconv.ParseInt(lineBits[1], 10, 0)
			//actualNumberInt64, err := strconv.ParseInt(lineBits[2], 10, 0)
			// tell them
			sendPublicMessage(conn, replyTo, theirNick, serverId, userId, "All your flag are belong to me! My badly optimized code did it in "+time.Since(startedTime).String()+", too.")
		} else if strings.HasPrefix(lineStr, "INVALID_URL") {
			sendPublicMessage(conn, replyTo, theirNick, serverId, userId, "'"+urlToUse+"' doesn't seem like a valid flag URL.")
			return
		}
	}
}

func main() {
	serverEightRegexp := regexp.MustCompile(`https://level08-([0-9]).stripe-ctf.com/user-([a-z]{10})/`)
	levelTwoUsername = "user-pnpgbrhmgp"
	levelTwoServer = "level02-4.stripe-ctf.com"
	usedMeMap = make(map[string]time.Time)
	currentUrls = make(map[string]bool)

	flag.Parse()

	c := irc.SimpleClient("FlagBot|lukegb")

	c.SSL = true

	c.AddHandler("connected", func(conn *irc.Conn, line *irc.Line) {
		log.Println("Connected - joining channels")
		conn.Join("#level8")
	})

	quit := make(chan bool)
	c.AddHandler("disconnected", func(conn *irc.Conn, line *irc.Line) {
		quit <- true
	})

	c.AddHandler("PRIVMSG", func(conn *irc.Conn, line *irc.Line) {
		// try matching the regexp
		lineData := strings.Join(line.Args[1:], " ")
		matches := serverEightRegexp.FindStringSubmatch(lineData)
		sentTo := line.Args[0]
		replyTo := line.Args[0]
		if sentTo[0] != ([]uint8("#"))[0] {
			sentTo = "__ME__"
			replyTo = line.Nick
		}
		//log.Printf("%s <%s> %s\n", line.Args[0], line.Nick, lineData)
		if matches != nil {
			meineUrl := fmt.Sprintf("https://level08-%s.stripe-ctf.com/user-%s/", matches[1], matches[2])
			log.Printf(" ---> FOUND URL: %s - using %s\n", matches[0], meineUrl)

			// am I running it already?
			canRun := lockUrl(meineUrl)
			if !canRun {
				log.Printf(" ---> Already running.\n")
				// ignore it
				return
			}

			// can they use me?
			canUse := lockUsedMe(line.Host)
			if line.Nick == "lukegb" || line.Nick == "lukegb_" {
				canUse = true
			}
			if !canUse {
				unlockUrl(meineUrl)
				log.Printf(" ---> Denied.\n")
				conn.Notice(line.Nick, "You've already pasted a URL - I'm not doing another one so fast.")
				return
			}

			// wooo
			conn.Privmsg(replyTo, fmt.Sprintf("%s: I'm off to break %s :)", line.Nick, meineUrl))
			go doTheBreak(conn, meineUrl, line.Nick, replyTo, matches[1], matches[2])
		}
	})

	if err := c.Connect("irc.stripe.com:6697"); err != nil {
		fmt.Printf("Connection error: %s\n", err.Error())
		os.Exit(1)
	}

	<-quit
}

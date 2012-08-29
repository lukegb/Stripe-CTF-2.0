package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	irc "github.com/fluffle/goirc/client"
	"io"
	"io/ioutil"
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
var serializeToDisk chan bool

type Flag struct {
	ServerId       string
	UserId         string
	BeganAt        time.Time
	Duration       time.Duration
	Result         int64
	Who            string
	WhoUserFoundIt bool
}

var knownFlags map[string]*Flag

func serializeToDiskLoop() {
	for {
		<-serializeToDisk
		log.Println("Beginning disk serialization")

		var marshaledFlags []byte
		var err error
		if marshaledFlags, err = json.Marshal(knownFlags); err != nil {
			log.Println("Error in marshalling: " + err.Error())
			continue
		}

		var file *os.File
		if file, err = os.Create("flagserialize.json"); err != nil {
			log.Println("Error in serialization: " + err.Error())
			continue
		}

		if _, err = file.Write(marshaledFlags); err != nil {
			log.Println("Error in serialization stage 2: " + err.Error())
			file.Close()
			continue
		}

		if err = file.Close(); err != nil {
			log.Println("Error closing file in serialization: " + err.Error())
			continue
		}

		log.Println("Ending disk serialization")
	}
}

func deserializeFromDisk(into interface{}) {
	var f *os.File
	var err error
	if f, err = os.Open("flagserialize.json"); err != nil {
		log.Println("Error deserializing (open): " + err.Error())
		return
	}
	defer f.Close()

	var dat []byte
	if dat, err = ioutil.ReadAll(f); err != nil {
		log.Println("Error deserializing (read): " + err.Error())
		return
	}

	if err = json.Unmarshal(dat, into); err != nil {
		log.Println("Error deserializing (unmarshal): " + err.Error())
	}
}

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

func doTheBreak(conn *irc.Conn, urlToUse string, theirNick string, replyTo string, serverId string, userId string, shouldSave bool) {
	defer unlockUrl(urlToUse)

	thisFlag := new(Flag)
	knownFlags[urlToUse] = thisFlag
	thisFlag.ServerId = serverId
	thisFlag.UserId = userId
	thisFlag.Who = theirNick

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
			thisFlag.BeganAt = time.Now()
		} else if strings.HasPrefix(lineStr, "BROKE ") {
			// now we tell them, but not much :)
			lineBits := strings.Split(lineStr, " ")
			chunkBrokenInt64, _ := strconv.ParseInt(lineBits[1], 10, 0)
			//actualNumberInt64, err := strconv.ParseInt(lineBits[2], 10, 0)
			// tell them
			sendPublicMessage(conn, replyTo, theirNick, serverId, userId, fmt.Sprintf("I've broken %d%% of your flag!", (chunkBrokenInt64+1)*25))
		} else if strings.HasPrefix(lineStr, "BROKE_ALL ") {
			// now we tell them, but not much :)
			lineBits := strings.Split(lineStr, " ")
			//chunkBrokenInt64, err := strconv.ParseInt(lineBits[1], 10, 0)
			var actualNumberInt64 int64
			var err error
			actualNumberInt64, err = strconv.ParseInt(lineBits[1], 10, 64)
			if err != nil {
				log.Println(err.Error())
				actualNumberInt64 = -1
			}
			// tell them
			thisFlag.Duration = time.Since(thisFlag.BeganAt)
			thisFlag.Result = actualNumberInt64
			serializeToDisk <- true
			sendPublicMessage(conn, replyTo, theirNick, serverId, userId, "All your flag are belong to me! My badly optimized code did it in "+strconv.FormatInt(int64(thisFlag.Duration.Seconds()), 10)+"s, too.")
		} else if strings.HasPrefix(lineStr, "INVALID_URL") {
			sendPublicMessage(conn, replyTo, theirNick, serverId, userId, "'"+urlToUse+"' doesn't seem like a valid flag URL.")
			return
		}
	}
}

func main() {
	serverEightRegexp := regexp.MustCompile(`https://level08-([0-9]).stripe-ctf.com/user-([a-z]{10})/`)
	serverEightBonusRegexp := regexp.MustCompile(`https://level8-bonus.danopia.net/([a-z0-9]{32})/`)
	flagNumberRegex := regexp.MustCompile(`[0-9]{12}`)
	levelTwoUsername = "user-pnpgbrhmgp"
	levelTwoServer = "level02-4.stripe-ctf.com"
	usedMeMap = make(map[string]time.Time)
	currentUrls = make(map[string]bool)

	// deserialize from disk
	knownFlags = make(map[string]*Flag)
	deserializeFromDisk(&knownFlags)

	serializeToDisk = make(chan bool)

	go serializeToDiskLoop()

	flag.Parse()

	c := irc.SimpleClient("FlagBot|lukegb")

	c.SSL = true

	c.AddHandler("connected", func(conn *irc.Conn, line *irc.Line) {
		log.Println("Connected - joining channels")
		conn.Join("#level8")
		conn.Join("#level8-bottest")
		conn.Join("#level8-bonus")
	})

	quit := make(chan bool)
	c.AddHandler("disconnected", func(conn *irc.Conn, line *irc.Line) {
		quit <- true
	})

	c.AddHandler("PRIVMSG", func(conn *irc.Conn, line *irc.Line) {
		// try matching the regexp
		lineData := strings.Join(line.Args[1:], " ")
		sentTo := line.Args[0]
		replyTo := line.Args[0]
		if sentTo[0] != ([]uint8("#"))[0] {
			sentTo = "__ME__"
			replyTo = line.Nick
		}
		//log.Printf("%s <%s> %s\n", line.Args[0], line.Nick, lineData)
		matches := serverEightRegexp.FindStringSubmatch(lineData)
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

			// now check to see if this is one I've already done
			flag, ok := knownFlags[meineUrl]
			if ok && !(strings.Contains(lineData, "FORCEGRAB") && (line.Nick == "lukegb" || line.Nick == "lukegb_")) {
				unlockUrl(meineUrl)
				// oh
				if flag.Result == 0 || flag.Result == -1 {
					log.Printf("Hmm, flag.Result was %d\n", flag.Result)
					// just keep going
				} else {
					conn.Privmsg(replyTo, fmt.Sprintf("%s: I've already done that flag. I did it in %ds, %s ago.", line.Nick, int64(flag.Duration.Seconds()), time.Since(flag.BeganAt.Add(flag.Duration)).String()))
					return
				}
			}

			// wooo
			conn.Privmsg(replyTo, fmt.Sprintf("%s: I'm off to break %s :)", line.Nick, meineUrl))
			go doTheBreak(conn, meineUrl, line.Nick, replyTo, matches[1], matches[2], true)
			return
		}

		// now try matching a l8-bonus round
		matches = serverEightBonusRegexp.FindStringSubmatch(lineData)
		if matches != nil {
			return
		}

		// try matching a flag response
		match := flagNumberRegex.FindString(lineData)
		if match != "" {
			mNum, err := strconv.ParseInt(match, 10, 64)
			if err != nil {
				conn.Privmsg("lukegb_", "Error parsing string "+match+" to int64 "+err.Error())
				return
			}
			for _, v := range knownFlags {
				if v.Result == mNum {
					if v.Who == line.Nick {
						if !v.WhoUserFoundIt {
							conn.Privmsg(replyTo, fmt.Sprintf("%s: Congrats on the capture :)", line.Nick))
							v.WhoUserFoundIt = true
							serializeToDisk <- true
						}
					} else {
						conn.Privmsg(replyTo, fmt.Sprintf("%s: ...was that flag actually yours? ._.", line.Nick))
					}
					break
				}
			}
			return
		}
	})

	if err := c.Connect("irc.stripe.com:6697"); err != nil {
		fmt.Printf("Connection error: %s\n", err.Error())
		os.Exit(1)
	}

	<-quit
}

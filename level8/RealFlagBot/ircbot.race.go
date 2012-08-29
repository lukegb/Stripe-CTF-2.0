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
	//"strconv"
	"strings"
	"time"
)

var levelTwoUsername string
var levelTwoServer string

var flagServ string

type Flag struct {
	BeganAt  time.Time
	Duration time.Duration
}

func doTheBreak(conn *irc.Conn, myCommand *exec.Cmd, stdoutPipe io.ReadCloser, urlToUse string) {
	thisFlag := new(Flag)

	// grab stdout
	var err error
	stdout := bufio.NewReader(stdoutPipe)
	if err != nil {
		log.Printf("Hit a snag whilst getting the stdoutpipe: %s\n", err.Error())
		conn.Privmsg("lukegb", "Oops - something went wrong finding the flag. Maybe some other time. Error getting pipe: "+err.Error())
		conn.Privmsg("#level8-bonus", "!leave")
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
			//actualNumberInt64, err := strconv.ParseInt(lineBits[2], 10, 0)
			// tell luke
			conn.Privmsg("lukegb", "Broke chunk: "+lineBits[1])

		} else if strings.HasPrefix(lineStr, "BROKE_ALL ") {
			lineBits := strings.Split(lineStr, " ")

			//sendPublicMessage(conn, replyTo, theirNick, serverId, userId, "All your flag are belong to me! My badly optimized code did it in "+strconv.FormatInt(int64(thisFlag.Duration.Seconds()), 10)+"s, too.")
			conn.Privmsg(flagServ, lineBits[1])
			break
		} else if strings.HasPrefix(lineStr, "INVALID_URL") {
			//sendPublicMessage(conn, replyTo, theirNick, serverId, userId, "'"+urlToUse+"' doesn't seem like a valid flag URL.")
			conn.Privmsg("lukegb", "I think FlagServ sent me an invalid URL: "+urlToUse)
			break
		}
	}
	myCommand.Wait()
	stdoutPipe.Close()
}

func main() {
	levelTwoUsername = "lukegb"
	levelTwoServer = "level2-bonus.danopia.net"
	flagServ = "FlagServ"

	flag.Parse()

	c := irc.SimpleClient("lukegbbot")

	c.SSL = true

	var myUrl string
	myUrl = ""

	var myCommand *exec.Cmd
	var myStdoutPipe io.ReadCloser
	var myStdinPipe io.WriteCloser

	c.AddHandler("connected", func(conn *irc.Conn, line *irc.Line) {
		log.Println("Connected - joining channels")
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
		//replyTo := line.Args[0]
		if sentTo[0] != ([]uint8("#"))[0] {
			sentTo = "__ME__"
			//replyTo = line.Nick
		}

		if sentTo != "__ME__" && line.Nick == flagServ && strings.Contains(lineData, "Go!") && myUrl != "" {
			// go!
			myStdinPipe.Write([]byte("\n"))
			go doTheBreak(conn, myCommand, myStdoutPipe, myUrl)
			myUrl = ""
		} else if sentTo != "__ME__" && lineData == "!start" {
			conn.Privmsg(sentTo, "!join")
		}

		if line.Nick == "lukegb" {
			// extra commands
			if strings.HasPrefix(lineData, "FB") {
				conn.Privmsg(sentTo, strings.Replace(lineData, "FB", "!", 1))
			}
		}

	})

	c.AddHandler("NOTICE", func(conn *irc.Conn, line *irc.Line) {
		// try matching the regexp
		lineData := strings.Join(line.Args[1:], " ")
		log.Println(line.Args)
		sentTo := line.Args[0]
		//replyTo := line.Args[0]
		if sentTo[0] != ([]uint8("#"))[0] {
			sentTo = "__ME__"
		}

		if sentTo == "__ME__" && line.Nick == flagServ {
			// ooh, it's a URL
			myUrl = strings.Replace(strings.Replace(lineData, "Your CTF endpoint will be <", "", 1), ">", "", 1)[1:67]
			myCommand = exec.Command("./pingit_recvit", "-servermode", myUrl, levelTwoServer)
			myStdoutPipe, _ = myCommand.StdoutPipe()
			myStdinPipe, _ = myCommand.StdinPipe()
			myCommand.Start()
		}

	})

	if err := c.Connect("irc.stripe.com:6697"); err != nil {
		fmt.Printf("Connection error: %s\n", err.Error())
		os.Exit(1)
	}

	<-quit
}

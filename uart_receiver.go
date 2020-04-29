// 12 august 2018

package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	serial "github.com/tarm/serial"
	serialv1 "go.bug.st/serial"
)

// BeaconInfo data of current connected beacon
type BeaconInfo struct {
	Address    string
	Initiator  bool
	BleEnabled bool
	NetworkID  int
	X          float64
	Y          float64
	Z          float64
}

// UartReceiver runs concurrent
type UartReceiver struct {
	PortName string
	Baud     int
	inBuff   [1024]byte
	inMsg    string
	serialP  *serial.Port
	IsSetUp  bool
	Data     BeaconInfo
	mux      sync.Mutex
}

//Connected returns weather uart is connected
func (ur *UartReceiver) Connected() bool {
	return ur.serialP != nil
}

// OpenPort establishs uart connection
func (ur *UartReceiver) OpenPort() bool {
	ur.IsSetUp = false
	c := &serial.Config{Name: ur.PortName, Baud: ur.Baud}
	// mode := &serial.Mode{
	// 	BaudRate: ur.Baud,
	// }

	s, err := serial.OpenPort(c)
	// p, err := serial.Open(ur.PortName, mode)
	if err != nil {
		log.Print("could not open com-port")
		log.Fatal(err)
		return false
	}
	ur.serialP = s
	go ur.receiverTask()
	return true
}

//ClosePort ...
func (ur *UartReceiver) ClosePort() {
	if ur.serialP != nil {
		ur.serialP.Close()
	}
	ur.serialP = nil
	ur.Data.Address = "?"
	ur.Data.Initiator = false
	ur.Data.BleEnabled = false
	ur.Data.NetworkID = 0
	ur.Data.X = 0.
	ur.Data.Y = 0.
	ur.Data.Z = 0.
}

//PortList string list of available ports
func (ur *UartReceiver) PortList() []string {
	ports, err := serialv1.GetPortsList()
	if err != nil {
		log.Fatal(err)
		return []string{}
	}
	if len(ports) == 0 {
		log.Print("No serial ports found!")
	}
	return ports
}

//SetPort sets the serial port, only affects new connection
func (ur *UartReceiver) SetPort(port string) {
	ur.PortName = port
}

// ReceiverTask goroutine for receiving bytes via uart connection
func (ur *UartReceiver) receiverTask() {
	for ur.Connected() {
		buf := make([]byte, 128)
		n, err := ur.serialP.Read(buf)
		if err != nil {
			log.Print(err)
			return
		}
		ur.inMsg += string(buf[:n])
		// log.Print(string(buf[:n]))
	}
}

// -------------------------------------------------
// request cmds

//RequestAll updates all info
func (ur *UartReceiver) RequestAll() bool {
	if !ur.SendStrAndWait("si\r") {
		ur.IsSetUp = false
	}
	if !ur.IsSetUp {
		if !ur.EnterShellMode() {
			return false
		}
	}
	if !ur.RequestSysInfo() {
		ur.IsSetUp = false
	}
	if !ur.RequestPos() {
		ur.IsSetUp = false
	}
	return ur.IsSetUp
}

// RequestSysInfo request and receive system info (network id)
func (ur *UartReceiver) RequestSysInfo() bool {
	if ur.SendStrAndWait("si\r") {
		if strings.Contains(ur.inMsg, "addr=x") {
			startI := strings.Index(ur.inMsg, "addr=x") + 7
			add := ur.inMsg[startI : startI+16]
			ur.Data.Address = add
		}
		if strings.Contains(ur.inMsg, "panid=x") {
			startI := strings.Index(ur.inMsg, "panid=x") + 7
			nID := ur.inMsg[startI : startI+4]
			netID, err := strconv.ParseUint(nID, 16, 64)
			if err != nil {
				log.Fatal(err)
			}
			ur.Data.NetworkID = int(netID)
		}
		if strings.Contains(ur.inMsg, "mode:") {
			if strings.Contains(ur.inMsg, "ani ") {
				ur.Data.Initiator = true
			}
			if strings.Contains(ur.inMsg, "an ") {
				ur.Data.Initiator = false
			}
		}
		if strings.Contains(ur.inMsg, "cfg:") {
			if strings.Contains(ur.inMsg, "ble=1") {
				ur.Data.BleEnabled = true
			}
			if strings.Contains(ur.inMsg, "ble=0") {
				ur.Data.BleEnabled = false
			}
		}
		return true
	}
	return false
}

// RequestPos request and receive positions
func (ur *UartReceiver) RequestPos() bool {
	if ur.SendStrAndWait("apg\r") {
		if strings.Contains(ur.inMsg, "apg:") {
			startI := strings.Index(ur.inMsg, "x:")
			parts := strings.Split(ur.inMsg[startI:], " ")
			if len(parts) >= 4 {
				x, e0 := strconv.ParseFloat(strings.ReplaceAll(parts[0], "x:", ""), 32)
				y, e1 := strconv.ParseFloat(strings.ReplaceAll(parts[1], "y:", ""), 32)
				z, e2 := strconv.ParseFloat(strings.ReplaceAll(parts[2], "z:", ""), 32)
				if e0 == nil && e1 == nil && e2 == nil {
					ur.Data.X = float64(x) / 1000
					ur.Data.Y = float64(y) / 1000
					ur.Data.Z = float64(z) / 1000
					return true
				}
			}
		}
	}
	return false
}

// -------------------------------------------------
// set cmds

// EnterShellMode changes from bin to shell mode
func (ur *UartReceiver) EnterShellMode() bool {
	if ur.SendStrAndWait("\r\r") {
		ur.IsSetUp = true
		return true
	}
	return false
}

//SetNetworkID sets beacons network id
func (ur *UartReceiver) SetNetworkID(newID int) bool {
	sendStr := fmt.Sprintf("nis %d\r", newID)
	if !ur.SendStrAndWaitForStr(sendStr, "nis: ok") {
		log.Print("failed to set networkID")
		ur.IsSetUp = false
		return false
	}
	ur.Data.NetworkID = newID
	if !ur.WaitForShellReadyNoReset() {
		log.Print("failed to wait for shell complete")
	}
	return true
}

//SetPosition sets position of beacon
func (ur *UartReceiver) SetPosition(x string, y string, z string) bool {
	xFl, err := strconv.ParseFloat(x, 64)
	if err != nil {
		log.Print("warning incalid value for x")
		ur.IsSetUp = false
		return false
	}
	yFl, err := strconv.ParseFloat(y, 64)
	if err != nil {
		log.Print("warning incalid value for y")
		ur.IsSetUp = false
		return false
	}
	zFl, err := strconv.ParseFloat(z, 64)
	if err != nil {
		log.Print("warning incalid value for z")
		ur.IsSetUp = false
		return false
	}

	sendStr := fmt.Sprintf("aps %d %d %d\r", int(xFl*1000), int(yFl*1000), int(zFl*1000))
	if !ur.SendStrAndWaitForStr(sendStr, "aps: ok") {
		log.Print("failed to set position")
		ur.IsSetUp = false
		return false
	}
	ur.Data.X = xFl
	ur.Data.Y = yFl
	ur.Data.Z = zFl
	if !ur.WaitForShellReadyNoReset() {
		log.Print("failed to wait for shell to complete")
	}
	return true
}

//SetMode sets beacon operating mode, and reboots to make it effictive
func (ur *UartReceiver) SetMode(initiator bool, enableBle bool) bool {
	inr := 0
	if initiator {
		inr = 1
	}
	bridge := 0
	enc := 0
	leds := 1
	ble := 0
	if enableBle {
		ble = 1
	}
	uwb := 2
	fwUpd := 0
	sendStr := fmt.Sprintf("acas %d %d %d %d %d %d %d\r",
		inr, bridge, enc, leds, ble, uwb, fwUpd)
	if !ur.SendStrAndWaitForStr(sendStr, "acas: ok") {
		log.Print("failed to set mode")
		ur.IsSetUp = false
		return false
	}
	if !ur.WaitForShellReadyNoReset() {
		log.Print("failed to wait for shell complete")
	}
	ur.SendStr("reset\r")
	ur.IsSetUp = false
	// device will reboot
	time.Sleep(700 * time.Millisecond)
	ur.RequestAll()
	return true
}

// -------------------------------------------------
// helper functions

// SendStrAndWait sends sentMsg and waits until dwm> is printed
func (ur *UartReceiver) SendStrAndWait(sentMsg string) bool {
	return ur.SendStrAndWaitForStr(sentMsg, "dwm>")
}

// SendStrAndWaitForStr sends sentMsg and waits until waitStr is printed
func (ur *UartReceiver) SendStrAndWaitForStr(sentMsg string, waitStr string) bool {
	ur.inMsg = ""
	if ur.serialP == nil {
		log.Print("no uart connection, abort send.")
		return false
	}
	if !ur.SendStr(sentMsg) {
		log.Print("failed to send msg to uart")
		return false
	}
	startS := time.Now()
	for !strings.Contains(ur.inMsg, waitStr) {
		if time.Now().Sub(startS) >= 2*time.Second {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
	return true
}

// WaitForShellReadyNoReset checks if dwm> already came or wait until it does
func (ur *UartReceiver) WaitForShellReadyNoReset() bool {
	startS := time.Now()
	for !strings.Contains(ur.inMsg, "dwm>") {
		if time.Now().Sub(startS) >= 2*time.Second {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
	return true
}

// SendStr sends sendMsg
func (ur *UartReceiver) SendStr(sentMsg string) bool {
	ur.mux.Lock()
	outBuff := []byte(sentMsg)
	for i := 0; i < len(outBuff); i++ {
		_, err := ur.serialP.Write([]byte{outBuff[i]})
		if err != nil {
			log.Fatal(err)
			ur.mux.Unlock()
			return false
		}
		// this is necessary because the in buffer of dwm is handled very bad
		time.Sleep(10 * time.Millisecond)
	}
	ur.mux.Unlock()
	return true
}

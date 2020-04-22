package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/andlabs/ui"
	_ "github.com/andlabs/ui/winmanifest"
)

//BeaconProgrammer holds all view elements that needs to be updated
type BeaconProgrammer struct {
	ComPortSelectCont *ui.Box
	ComPortCombo      *ui.Combobox
	RefreshPortsBut   *ui.Button
	ConnectBut        *ui.Button
	DeviceAddLab      *ui.Label
	NetworkIDEntry    *ui.Entry
	NetworkIDHexLab   *ui.Label
	NetworkIDHexChk   *ui.Checkbox
	ModeRb            *ui.RadioButtons
	BleChk            *ui.Checkbox
	PosXEntry         *ui.Entry
	PosYEntry         *ui.Entry
	PosZEntry         *ui.Entry
	StatProgBar       *ui.ProgressBar
	PortList          []string
	StatusLab         *ui.Label
	ResetBut          *ui.Button
	SaveBut           *ui.Button
}

var mainwin *ui.Window
var ur UartReceiver
var bp BeaconProgrammer

func makeBasicControlsPage() ui.Control {
	vbox := ui.NewVerticalBox()
	vbox.SetPadded(true)

	comPortGrid := ui.NewGrid()
	comPortGrid.SetPadded(true)
	vbox.Append(comPortGrid, false)

	//***************************************
	// connection
	comForm := ui.NewForm()
	comForm.SetPadded(true)
	comPortGrid.Append(ui.NewLabel("com-port"),
		0, 0, 1, 1,
		false, ui.AlignFill, false, ui.AlignFill)

	bp.ComPortSelectCont = ui.NewHorizontalBox()
	bp.ComPortCombo = ui.NewCombobox()
	bp.ComPortSelectCont.Append(bp.ComPortCombo, false)
	refreshComPorts()
	comPortGrid.Append(bp.ComPortSelectCont,
		1, 0, 1, 1,
		false, ui.AlignFill, false, ui.AlignFill)

	//comPortGrid.Append(cbox, false)
	bp.RefreshPortsBut = ui.NewButton("refresh")
	bp.RefreshPortsBut.OnClicked(refreshComPortsCallback)
	// refreshPortsBut.OnClicked(resetButton)
	comPortGrid.Append(bp.RefreshPortsBut,
		2, 0, 1, 1,
		false, ui.AlignFill, false, ui.AlignFill)
	bp.ConnectBut = ui.NewButton("connect")
	bp.ConnectBut.OnClicked(connectCallback)
	comPortGrid.Append(bp.ConnectBut,
		3, 0, 1, 1,
		false, ui.AlignFill, false, ui.AlignFill)
	//(refreshPortsBut, false)
	vbox.Append(comForm, false)
	//groupCom.SetChild(comForm)

	vbox.Append(ui.NewLabel("set up beacon..."), false)
	vbox.Append(ui.NewHorizontalSeparator(), false)

	//***************************************
	// meta
	groupMeta := ui.NewGroup("Meta")
	groupMeta.SetMargined(true)
	vbox.Append(groupMeta, true)
	metaForm := ui.NewForm()
	metaForm.SetPadded(true)
	groupMeta.SetChild(metaForm)

	bp.DeviceAddLab = ui.NewLabel("?")
	metaForm.Append("DeviceAddress", bp.DeviceAddLab, false)

	networkIDGrid := ui.NewGrid()
	networkIDGrid.SetPadded(true)
	metaForm.Append("NetworkID", networkIDGrid, false)
	bp.NetworkIDEntry = ui.NewEntry()
	networkIDGrid.Append(bp.NetworkIDEntry,
		0, 0, 1, 1,
		false, ui.AlignFill, false, ui.AlignFill)
	bp.NetworkIDHexChk = ui.NewCheckbox("use HEX")
	bp.NetworkIDHexChk.OnToggled(toggleNetworIDHexChk)
	networkIDGrid.Append(bp.NetworkIDHexChk,
		1, 0, 1, 1,
		false, ui.AlignFill, false, ui.AlignFill)
	bp.NetworkIDHexLab = ui.NewLabel("hex: ?")
	// networkIDGrid.Append(bp.NetworkIDHexLab,
	// 	0, 1, 1, 1,
	// 	false, ui.AlignFill, false, ui.AlignFill)
	bp.ModeRb = ui.NewRadioButtons()
	bp.ModeRb.Append("normal")
	bp.ModeRb.Append("initiator")
	metaForm.Append("mode", bp.ModeRb, false)

	bp.BleChk = ui.NewCheckbox("")
	metaForm.Append("enable BLE", bp.BleChk, false)

	//***************************************
	// position
	groupPos := ui.NewGroup("Position")
	groupPos.SetMargined(true)
	vbox.Append(groupPos, true)
	posForm := ui.NewForm()
	posForm.SetPadded(true)
	groupPos.SetChild(posForm)
	// posForm.Append("x in m", ui.NewSpinbox(-9999999, 9999999), false)
	// posForm.Append("y in m", ui.NewSpinbox(0, 100), false)
	// posForm.Append("z in m", ui.NewSpinbox(0, 100), false)

	bp.PosXEntry = ui.NewEntry()
	posForm.Append("x in m", bp.PosXEntry, false)
	bp.PosYEntry = ui.NewEntry()
	posForm.Append("y in m", bp.PosYEntry, false)
	bp.PosZEntry = ui.NewEntry()
	posForm.Append("z in m", bp.PosZEntry, false)

	hbox := ui.NewHorizontalBox()
	hbox.SetPadded(true)
	vbox.Append(hbox, false)
	bp.ResetBut = ui.NewButton("download")
	bp.ResetBut.OnClicked(resetButtonCallback)
	hbox.Append(bp.ResetBut, false)
	bp.SaveBut = ui.NewButton("save")
	bp.SaveBut.OnClicked(saveButtonCallback)
	hbox.Append(bp.SaveBut, false)

	bp.StatProgBar = ui.NewProgressBar()
	bp.StatProgBar.SetValue(0)
	vbox.Append(bp.StatProgBar, false)

	bp.StatusLab = ui.NewLabel("disconnected!")
	vbox.Append(bp.StatusLab, false)
	enableAll(false)
	return vbox
}

func startOperation(msg string) {
	enableAll(false)
	bp.StatusLab.SetText(msg)
	bp.StatProgBar.SetValue(-1)
}

func doneOperation(msg string) {
	bp.StatusLab.SetText(msg)
	bp.StatProgBar.SetValue(0)
	if ur.Connected() {
		enableAll(true)
	}
}

func enableAll(enable bool) {
	if enable {
		bp.NetworkIDEntry.Enable()
		bp.NetworkIDHexChk.Enable()
		bp.ModeRb.Enable()
		bp.BleChk.Enable()
		bp.PosXEntry.Enable()
		bp.PosYEntry.Enable()
		bp.PosZEntry.Enable()
		bp.ResetBut.Enable()
		bp.SaveBut.Enable()
	} else {
		bp.NetworkIDEntry.Disable()
		bp.NetworkIDHexChk.Disable()
		bp.ModeRb.Disable()
		bp.BleChk.Disable()
		bp.PosXEntry.Disable()
		bp.PosYEntry.Disable()
		bp.PosZEntry.Disable()
		bp.ResetBut.Disable()
		bp.SaveBut.Disable()
	}
}

func refreshComPorts() {
	bp.PortList = ur.PortList()
	bp.ComPortSelectCont.Delete(0)
	bp.ComPortCombo = ui.NewCombobox()
	bp.ComPortSelectCont.Append(bp.ComPortCombo, false)
	for _, p := range bp.PortList {
		bp.ComPortCombo.Append(p)
	}
	if len(bp.PortList) > 0 {
		bp.ComPortCombo.SetSelected(0)
	}
}

func refreshView() {
	if ur.Connected() {
		bp.ConnectBut.SetText("Disconnect")
		bp.ComPortCombo.Disable()
		bp.RefreshPortsBut.Disable()
	} else {
		bp.ConnectBut.SetText("Connect")
		bp.ComPortCombo.Enable()
		bp.RefreshPortsBut.Enable()
	}
	bp.DeviceAddLab.SetText(ur.Data.Address)
	if bp.NetworkIDHexChk.Checked() {
		bp.NetworkIDEntry.SetText(fmt.Sprintf("%x", ur.Data.NetworkID))
		bp.NetworkIDHexLab.SetText(fmt.Sprintf("dec: %d", ur.Data.NetworkID))
	} else {
		bp.NetworkIDEntry.SetText(strconv.Itoa(ur.Data.NetworkID))
		bp.NetworkIDHexLab.SetText(fmt.Sprintf("hex: %x", ur.Data.NetworkID))
	}
	selectedIndex := 0
	if ur.Data.Initiator {
		selectedIndex = 1
	}
	bp.ModeRb.SetSelected(selectedIndex)
	bp.BleChk.SetChecked(ur.Data.BleEnabled)
	bp.PosXEntry.SetText(strconv.FormatFloat(ur.Data.X, 'f', 3, 64))
	bp.PosYEntry.SetText(strconv.FormatFloat(ur.Data.Y, 'f', 3, 64))
	bp.PosZEntry.SetText(strconv.FormatFloat(ur.Data.Z, 'f', 3, 64))
}

func refreshComPortsCallback(but *ui.Button) {
	refreshComPorts()
}

func connect() {
	ur.SetPort(bp.PortList[bp.ComPortCombo.Selected()])
	ur.OpenPort()
	doneOperation("connected!")
	startOperation("reading data...")
	downloadData()
}

func disconnect() {
	ur.ClosePort()
	refreshView()
	doneOperation("disconnected!")
}

func connectCallback(but *ui.Button) {
	if ur.Connected() {
		startOperation("disconnecting...")
		go disconnect()
	} else {
		if len(bp.PortList) > 0 {
			startOperation("connecting...")
			go connect()
		}
	}
}

func downloadData() {
	suc := ur.RequestAll()
	refreshView()
	if suc {
		doneOperation("done")
		return
	}
	doneOperation("error, could not read from device")
}

func resetButtonCallback(but *ui.Button) {
	startOperation("reading data...")
	go downloadData()
}

func saveData() {
	if bp.NetworkIDHexChk.Checked() {
		netID, err := strconv.ParseUint(bp.NetworkIDEntry.Text(), 16, 64)
		if err != nil {
			log.Fatal(err)
		}
		ur.SetNetworkID(int(netID))
	} else {
		netID, err := strconv.Atoi(bp.NetworkIDEntry.Text())
		if err != nil {
			log.Print(err)
			doneOperation("an error accured")
			return
		}
		ur.SetNetworkID(netID)
	}
	ur.SetPosition(bp.PosXEntry.Text(), bp.PosYEntry.Text(), bp.PosZEntry.Text())
	init := false
	if bp.ModeRb.Selected() == 1 {
		init = true
	}
	ur.SetMode(init, bp.BleChk.Checked())
	refreshView()
	doneOperation("done")
}

func saveButtonCallback(but *ui.Button) {
	startOperation("saving...")
	go saveData()
}

func toggleNetworIDHexChk(chk *ui.Checkbox) {
	refreshView()
}

func setupUI() {
	mainwin = ui.NewWindow("Beacon-Programmer", 40, 40, true)
	mainwin.OnClosing(func(*ui.Window) bool {
		ui.Quit()
		ur.ClosePort()
		return true
	})
	ui.OnShouldQuit(func() bool {
		mainwin.Destroy()
		return true
	})
	mainwin.SetChild(makeBasicControlsPage())
	mainwin.SetMargined(true)
	mainwin.Show()
}

func main() {
	ur = UartReceiver{PortName: "COM6", Baud: 115200}
	ui.Main(setupUI)
}

/*
Package serialport provides a binding to libserialport for serial
port functionality.

Example Usage

  package main

  import (
    "github.com/mikepb/go-serialport"
    "log"
  )

  func main() {
    p := serialport.Open("/dev/tty")

    // optional, will automatically close when garbage collected
    defer p.Close()

    if err := p.SetBaudrate(115200) {
      log.Panic(err)
    }

    var buf []byte
    if c, err := p.Read(buf); err != nil {
      log.Panic(err)
    } else {
      log.Println(buf)
    }
  }
*/
package serialport

/*
#cgo CFLAGS:  -g -O2 -Wall -Wextra -DSP_PRIV= -DSP_API=
#cgo linux pkg-config: -ludev
#cgo darwin LDFLAGS: -framework IOKit -framework CoreFoundation
#include <stdlib.h>
#include "libserialport.h"
*/
import "C"

import (
	"errors"
	"os"
	"runtime"
	"unsafe"
)

const (
	// Port access modes
	MODE_READ  = C.SP_MODE_READ  // Open port for read access
	MODE_WRITE = C.SP_MODE_WRITE // Open port for write access

	// Port events.
	EVENT_RX_READY = C.SP_EVENT_RX_READY // Data received and ready to read.
	EVENT_TX_READY = C.SP_EVENT_TX_READY // Ready to transmit new data.
	EVENT_ERROR    = C.SP_EVENT_ERROR    // Error occured.

	// Buffer selection.
	BUF_INPUT  = C.SP_BUF_INPUT  // Input buffer.
	BUF_OUTPUT = C.SP_BUF_OUTPUT // Output buffer.
	BUF_BOTH   = C.SP_BUF_BOTH   // Both buffers.

	// Parity settings.
	PARITY_INVALID = C.SP_PARITY_INVALID // Special value to indicate setting should be left alone.
	PARITY_NONE    = C.SP_PARITY_NONE    // No parity.
	PARITY_ODD     = C.SP_PARITY_ODD     // Odd parity.
	PARITY_EVEN    = C.SP_PARITY_EVEN    // Even parity.
	PARITY_MARK    = C.SP_PARITY_MARK    // Mark parity.
	PARITY_SPACE   = C.SP_PARITY_SPACE   // Space parity.

	// RTS pin behaviour.
	RTS_INVALID      = C.SP_RTS_INVALID      // Special value to indicate setting should be left alone.
	RTS_OFF          = C.SP_RTS_OFF          // RTS off.
	RTS_ON           = C.SP_RTS_ON           // RTS on.
	RTS_FLOW_CONTROL = C.SP_RTS_FLOW_CONTROL // RTS used for flow control.

	// CTS pin behaviour.
	CTS_INVALID      = C.SP_CTS_INVALID      // Special value to indicate setting should be left alone.
	CTS_IGNORE       = C.SP_CTS_IGNORE       // CTS ignored.
	CTS_FLOW_CONTROL = C.SP_CTS_FLOW_CONTROL // CTS used for flow control.

	// DTR pin behaviour.
	DTR_INVALID      = C.SP_DTR_INVALID      // Special value to indicate setting should be left alone.
	DTR_OFF          = C.SP_DTR_OFF          // DTR off.
	DTR_ON           = C.SP_DTR_ON           // DTR on.
	DTR_FLOW_CONTROL = C.SP_DTR_FLOW_CONTROL // DTR used for flow control.

	// DSR pin behaviour.
	DSR_INVALID      = C.SP_DSR_INVALID      // Special value to indicate setting should be left alone.
	DSR_IGNORE       = C.SP_DSR_IGNORE       // DSR ignored.
	DSR_FLOW_CONTROL = C.SP_DSR_FLOW_CONTROL // DSR used for flow control.

	// XON/XOFF flow control behaviour.
	XONXOFF_INVALID  = C.SP_XONXOFF_INVALID  // Special value to indicate setting should be left alone.
	XONXOFF_DISABLED = C.SP_XONXOFF_DISABLED // XON/XOFF disabled.
	XONXOFF_IN       = C.SP_XONXOFF_IN       // XON/XOFF enabled for input only.
	XONXOFF_OUT      = C.SP_XONXOFF_OUT      // XON/XOFF enabled for output only.
	XONXOFF_INOUT    = C.SP_XONXOFF_INOUT    // XON/XOFF enabled for input and output.

	// Standard flow control combinations.
	FLOWCONTROL_NONE    = C.SP_FLOWCONTROL_NONE    // No flow control.
	FLOWCONTROL_XONXOFF = C.SP_FLOWCONTROL_XONXOFF // Software flow control using XON/XOFF characters.
	FLOWCONTROL_RTSCTS  = C.SP_FLOWCONTROL_RTSCTS  // Hardware flow control using RTS/CTS signals.
	FLOWCONTROL_DTRDSR  = C.SP_FLOWCONTROL_DTRDSR  // Hardware flow control using DTR/DSR signals.

	// Input signals
	SIG_CTS = C.SP_SIG_CTS // Clear to send
	SIG_DSR = C.SP_SIG_DSR // Data set ready
	SIG_DCD = C.SP_SIG_DCD // Data carrier detect
	SIG_RI  = C.SP_SIG_RI  // Ring indicator

	// Transport types.
	TRANSPORT_NATIVE    = C.SP_TRANSPORT_NATIVE    // Native platform serial port.
	TRANSPORT_USB       = C.SP_TRANSPORT_USB       // USB serial port adapter.
	TRANSPORT_BLUETOOTH = C.SP_TRANSPORT_BLUETOOTH // Bluetooh serial port adapter.
)

type SerialPort struct {
	name   string
	p      *C.struct_sp_port
	c      *C.struct_sp_port_config
	f      *os.File
	fd     uintptr
	opened bool
}

var InvalidArgumentsError = errors.New("Invalid arguments were passed to the function")
var SystemError = errors.New("A system error occured while executing the operation")
var MemoryAllocationError = errors.New("A memory allocation failed while executing the operation")
var UnsupportedOperationError = errors.New("The requested operation is not supported by this system or device")

// Map error codes to errors.
func errmsg(err C.enum_sp_return) error {
	switch err {
	case C.SP_ERR_ARG:
		return InvalidArgumentsError
	case C.SP_ERR_FAIL:
		return SystemError
	case C.SP_ERR_MEM:
		return MemoryAllocationError
	case C.SP_ERR_SUPP:
		return UnsupportedOperationError
	}
	return nil
}

// Wrap a sp_port struct in a go SerialPort struct and set finalizer for
// garbage collection.
func newSerialPort(p *C.struct_sp_port) (*SerialPort, error) {
	sp := &SerialPort{p: p}
	if err := errmsg(C.sp_new_config(&sp.c)); err != nil {
		return nil, err
	}
	runtime.SetFinalizer(sp, (*SerialPort).free)
	return sp, nil
}

// Finalizer callback for garbage collection.
func (p *SerialPort) free() {
	if p.opened {
		C.sp_close(p.p)
	}
	if p.p != nil {
		C.sp_free_port(p.p)
	}
	if p.c != nil {
		C.sp_free_config(p.c)
	}
	p.opened = false
	p.p = nil
	p.c = nil
}

// Get a port by name. The returned port is not opened automatically;
// use p.Open() or p.OpenWithMode(...) to open the port for I/O.
func PortByName(name string) (*SerialPort, error) {
	var p *C.struct_sp_port

	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))

	if err := errmsg(C.sp_get_port_by_name(cname, &p)); err != nil {
		return nil, err
	}

	return newSerialPort(p)
}

// List the serial ports available on the system. The returned ports
// are not opened automatically; use p.Open() or p.OpenWithMode(...)
// to open the port(s) for I/O.
func ListPorts() ([]*SerialPort, error) {
	var p **C.struct_sp_port

	if err := C.sp_list_ports(&p); err != C.SP_OK {
		return nil, errmsg(err)
	}
	defer C.sp_free_port_list(p)

	pp := (*[1 << 30]*C.struct_sp_port)(unsafe.Pointer(p))

	// count number of ports
	c := 0
	for ; uintptr(unsafe.Pointer(pp[c])) != 0; c++ {
	}

	// populate
	ports := make([]*SerialPort, c)
	for j := 0; j < c; j++ {
		var pc *C.struct_sp_port
		if err := errmsg(C.sp_copy_port(pp[j], &pc)); err != nil {
			return nil, err
		}
		if sp, err := newSerialPort(pc); err != nil {
			return nil, err
		} else {
			ports[j] = sp
		}
	}

	return ports, nil
}

// Get the name of a port.
func (p *SerialPort) Name() string {
	return C.GoString(C.sp_get_port_name(p.p))
}

// Get a description for a port, to present to end user.
func (p *SerialPort) Description() string {
	return C.GoString(C.sp_get_port_description(p.p))
}

// Get the transport type used by a port.
func (p *SerialPort) Transport() int {
	t := C.sp_get_port_transport(p.p)
	return int(t)
}

// Get the USB bus number and address on bus of a USB serial adapter port.
func (p *SerialPort) USBBusAddress() (int, int, error) {
	var bus, address C.int
	if err := errmsg(C.sp_get_port_usb_bus_address(p.p, &bus, &address)); err != nil {
		return 0, 0, err
	}
	return int(bus), int(address), nil
}

// Get the USB Vendor ID and Product ID of a USB serial adapter port.
func (p *SerialPort) USBVIDPID() (int, int, error) {
	var vid, pid C.int
	if err := errmsg(C.sp_get_port_usb_vid_pid(p.p, &vid, &pid)); err != nil {
		return 0, 0, err
	}
	return int(vid), int(pid), nil
}

// Get the USB manufacturer string of a USB serial adapter port.
func (p *SerialPort) USBManufacturer() string {
	cdesc := C.sp_get_port_usb_manufacturer(p.p)
	return C.GoString(cdesc)
}

// Get the USB product string of a USB serial adapter port.
func (p *SerialPort) USBProduct() string {
	cdesc := C.sp_get_port_usb_product(p.p)
	return C.GoString(cdesc)
}

// Get the USB serial number string of a USB serial adapter port.
func (p *SerialPort) USBSerialNumber() string {
	cdesc := C.sp_get_port_usb_serial(p.p)
	return C.GoString(cdesc)
}

// Get the MAC address of a Bluetooth serial adapter port.
func (p *SerialPort) BluetoothAddress() string {
	cdesc := C.sp_get_port_bluetooth_address(p.p)
	return C.GoString(cdesc)
}

// Open a port by name. Same as calling:
//   p := serialport.PortByName(name)
//   serialport.Open(serialport.MODE_READ|serialport.MODE_WRITE)
func Open(name string) (*SerialPort, error) {
	p, err := PortByName(name)
	if err != nil {
		return nil, err
	}
	err = p.Open()
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Open the serial port in read/write mode.
func (p *SerialPort) Open() error {
	return p.OpenWithMode(MODE_READ | MODE_WRITE)
}

// Open the serial port with the given mode.
func (p *SerialPort) OpenWithMode(flags uint) (err error) {
	if err = errmsg(C.sp_open(p.p, C.enum_sp_mode(flags))); err != nil {
		return
	}
	if err = errmsg(C.sp_get_config(p.p, p.c)); err != nil {
		p.Close()
		return
	}
	if err = errmsg(C.sp_get_port_handle(p.p, unsafe.Pointer(&p.fd))); err != nil {
		p.Close()
		return
	}
	p.f = os.NewFile(p.fd, p.Name())
	p.opened = true
	return nil
}

// Close the serial port.
func (p *SerialPort) Close() error {
	err := errmsg(C.sp_close(p.p))
	p.opened = false
	return err
}

// Get the baud rate from a port configuration. The port must be
// opened for this operation.
func (p *SerialPort) Baudrate() (int, error) {
	var baudrate C.int
	if err := errmsg(C.sp_get_config_baudrate(p.c, &baudrate)); err != nil {
		return 0, err
	}
	return int(baudrate), nil
}

// Set the baud rate for the serial port. The port must be opened for
// this operation. Call p.ApplyConfig() to apply the change.
func (p *SerialPort) SetBaudrate(baudrate int) error {
	return errmsg(C.sp_set_baudrate(p.p, C.int(baudrate)))
}

// Get the data bits from a port configuration. The port must be
// opened for this operation.
func (p *SerialPort) DataBits() (int, error) {
	var bits C.int
	if err := errmsg(C.sp_get_config_bits(p.c, &bits)); err != nil {
		return 0, err
	}
	return int(bits), nil
}

// Set the number of data bits for the serial port. The port must be
// opened for this operation. Call p.ApplyConfig() to apply the
// change.
func (p *SerialPort) SetDataBits(bits int) error {
	return errmsg(C.sp_set_config_bits(p.c, C.int(bits)))
}

// Get the parity setting from a port configuration. The port must be
// opened for this operation.
func (p *SerialPort) Parity() (int, error) {
	var parity C.enum_sp_return
	if err := errmsg(C.sp_get_config_parity(p.c, &parity)); err != nil {
		return 0, err
	}
	return int(parity), nil
}

// Set the parity setting for the serial port. The port must be opened
// for this operation. Call p.ApplyConfig() to apply the change.
func (p *SerialPort) SetParity(parity int) error {
	return errmsg(C.sp_set_config_parity(p.c, C.enum_sp_return(parity)))
}

// Get the stop bits from a port configuration. The port must be
// opened for this operation.
func (p *SerialPort) StopBits() (int, error) {
	var stopbits C.int
	if err := errmsg(C.sp_get_config_stopbits(p.c, &stopbits)); err != nil {
		return 0, err
	}
	return int(stopbits), nil
}

// Set the stop bits for the serial port. The port must be opened for
// this operation. Call p.ApplyConfig() to apply the change.
func (p *SerialPort) SetStopBits(stopbits int) error {
	return errmsg(C.sp_set_config_stopbits(p.c, C.int(stopbits)))
}

// Get the RTS pin behaviour from a port configuration. The port must
// be opened for this operation.
func (p *SerialPort) RTS() (int, error) {
	var rts C.enum_sp_rts
	if err := errmsg(C.sp_get_config_rts(p.c, &rts)); err != nil {
		return 0, err
	}
	return int(rts), nil
}

// Set the RTS pin behaviour in a port configuration. The port must be
// opened for this operation. Call p.ApplyConfig() to apply the
// change.
func (p *SerialPort) SetRTS(rts int) error {
	return errmsg(C.sp_set_config_rts(p.c, C.enum_sp_rts(rts)))
}

// Get the CTS pin behaviour from a port configuration. The port must
// be opened for this operation.
func (p *SerialPort) CTS() (int, error) {
	var cts C.enum_sp_cts
	if err := errmsg(C.sp_get_config_cts(p.c, &cts)); err != nil {
		return 0, err
	}
	return int(cts), nil
}

// Set the CTS pin behaviour in a port configuration. The port must be
// opened for this operation. Call p.ApplyConfig() to apply the
// change.
func (p *SerialPort) SetCTS(cts int) error {
	return errmsg(C.sp_set_config_cts(p.c, C.enum_sp_cts(cts)))
}

// Get the DTR pin behaviour from a port configuration. The port must
// be opened for this operation.
func (p *SerialPort) DTR() (int, error) {
	var dtr C.enum_sp_rts
	if err := errmsg(C.sp_get_config_dtr(p.c, &dtr)); err != nil {
		return 0, err
	}
	return int(dtr), nil
}

// Set the DTR pin behaviour in a port configuration. The port must be
// opened for this operation. Call p.ApplyConfig() to apply the
// change.
func (p *SerialPort) SetDTR(dtr int) error {
	return errmsg(C.sp_set_config_dtr(p.c, C.enum_sp_dtr(dtr)))
}

// Get the DSR pin behaviour from a port configuration. The port must
// be opened for this operation.
func (p *SerialPort) DSR() (int, error) {
	var dsr C.enum_sp_dsr
	if err := errmsg(C.sp_get_config_dsr(p.c, &dsr)); err != nil {
		return 0, err
	}
	return int(dsr), nil
}

// Set the DSR pin behaviour in a port configuration. The port must be
// opened for this operation. Call p.ApplyConfig() to apply the
// change.
func (p *SerialPort) SetDSR(dsr int) error {
	return errmsg(C.sp_set_config_dsr(p.c, C.enum_sp_dsr(dsr)))
}

// Get the XON/XOFF configuration from a port configuration. The port
// must be opened for this operation.
func (p *SerialPort) XonXoff() (int, error) {
	var xon C.enum_sp_xonxoff
	if err := errmsg(C.sp_get_config_xon_xoff(p.c, &xon)); err != nil {
		return 0, err
	}
	return int(xon), nil
}

// Set the XON/XOFF configuration in a port configuration. The port
// must be opened for this operation. Call p.ApplyConfig() to apply
// the change.
func (p *SerialPort) SetXonXoff(xon int) error {
	return errmsg(C.sp_set_config_xon_xoff(p.c, C.enum_sp_xonxoff(xon)))
}

// Set the flow control type in a port configuration. The port must be
// opened for this operation. Call p.ApplyConfig() to apply the
// change.
func (p *SerialPort) SetFlowControl(xon int) error {
	return errmsg(C.sp_set_config_flowcontrol(p.c, C.enum_sp_flowcontrol(xon)))
}

// Apply the configuration for the serial port.
func (p *SerialPort) ApplyConfig() error {
	return errmsg(C.sp_set_config(p.p, p.c))
}

// Apply the raw mode configuration for the serial port.
func (p *SerialPort) ApplyRawConfig() (err error) {
	if err = p.SetDataBits(8); err != nil {
		return
	}
	if err = p.SetParity(PARITY_NONE); err != nil {
		return
	}
	if err = p.SetStopBits(1); err != nil {
		return
	}
	if err = p.SetRTS(RTS_OFF); err != nil {
		return
	}
	if err = p.SetCTS(CTS_IGNORE); err != nil {
		return
	}
	if err = p.SetDTR(DTR_OFF); err != nil {
		return
	}
	if err = p.SetDSR(DSR_IGNORE); err != nil {
		return
	}
	if err = p.SetXonXoff(XONXOFF_DISABLED); err != nil {
		return
	}
	if err = p.SetFlowControl(FLOWCONTROL_NONE); err != nil {
		return
	}
	return errmsg(C.sp_set_config(p.p, p.c))
}

// Implementation of io.Reader interface.
func (p *SerialPort) Read(b []byte) (int, error) {
	return p.f.Read(b)
}

// Implementation of io.Writer interface.
func (p *SerialPort) Write(b []byte) (int, error) {
	return p.f.Write(b)
}

// WriteString is like Write, but writes the contents of string s
// rather than a slice of bytes.
func (p *SerialPort) WriteString(s string) (int, error) {
	return p.f.WriteString(s)
}

// Sync commits the current contents of the file to stable storage.
// Typically, this means flushing the file system's in-memory copy of
// recently written data to disk.
func (p *SerialPort) Sync() error {
	return p.f.Sync()
}

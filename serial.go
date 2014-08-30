/*

Package serial provides a binding to libserialport for serial port
functionality. Serial ports are commonly used with embedded systems,
such as the Arduino platform.

Example Usage

  package main

  import (
    "github.com/mikepb/go-serial"
    "log"
  )

  func main() {
    p, err := serial.Open("/dev/tty",
      serial.Options{Baudrate: 115200})
    if err != nil {
      log.Panic(err)
    }

    // optional, will automatically close when garbage collected
    defer p.Close()

    buf := make([]byte, 1)
    if c, err := p.Read(buf); err != nil {
      log.Panic(err)
    } else {
      log.Println(buf)
    }
  }

*/
package serial

/*
#cgo CFLAGS: -g -O2 -Wall -Wextra -DSP_PRIV= -DSP_API=
#cgo darwin LDFLAGS: -framework IOKit -framework CoreFoundation
#include <stdlib.h>
#include "libserialport.h"
*/
import "C"

import (
	"errors"
	"os"
	"reflect"
	"runtime"
	"time"
	"unsafe"
)

// Serial port options.
type Options struct {
	Mode        int // read, write; default is read/write
	Baudrate    int // number of bits per second (baudrate); default is 9600
	DataBits    int // number of data bits (5, 6, 7, 8); default is 8
	StopBits    int // number of stop bits (1, 2); default is 1
	Parity      int // none, odd, even, mark, space; default is none
	FlowControl int // none, xonxoff, rtscts, dtrdsr; default is none.
}

const (
	// Port access modes
	MODE_READ  = C.SP_MODE_READ  // Open port for read access
	MODE_WRITE = C.SP_MODE_WRITE // Open port for write access

	// Port events.
	EVENT_RX_READY = C.SP_EVENT_RX_READY // Data received and ready to read.
	EVENT_TX_READY = C.SP_EVENT_TX_READY // Ready to transmit new data.
	EVENT_ERROR    = C.SP_EVENT_ERROR    // Error occured.

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

// Serial port.
type Port struct {
	name          string
	p             *C.struct_sp_port
	c             *C.struct_sp_port_config
	f             *os.File
	fd            uintptr
	opened        bool
	readDeadline  time.Time
	writeDeadline time.Time
}

// Implementation of net.Addr
type Addr struct {
	name string
}

// Implementation of net.Addr.Network()
func (a *Addr) Network() string {
	return a.name
}

// Implementation of net.Addr.String()
func (a *Addr) String() string {
	return a.name
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

// Wrap a sp_port struct in a go Port struct and set finalizer for
// garbage collection.
func newSerialPort(p *C.struct_sp_port) (*Port, error) {
	sp := &Port{p: p}
	if err := errmsg(C.sp_new_config(&sp.c)); err != nil {
		return nil, err
	}
	runtime.SetFinalizer(sp, (*Port).free)
	return sp, nil
}

// Finalizer callback for garbage collection.
func (p *Port) free() {
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
func PortByName(name string) (*Port, error) {
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
func ListPorts() ([]*Port, error) {
	var p **C.struct_sp_port

	if err := C.sp_list_ports(&p); err != C.SP_OK {
		return nil, errmsg(err)
	}
	defer C.sp_free_port_list(p)

	// Convert the C array into a Go slice
	// See: https://code.google.com/p/go-wiki/wiki/cgo
	pp := (*[1 << 15]*C.struct_sp_port)(unsafe.Pointer(p))

	// count number of ports
	c := 0
	for ; uintptr(unsafe.Pointer(pp[c])) != 0; c++ {
	}

	// populate
	ports := make([]*Port, c)
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
func (p *Port) Name() string {
	return C.GoString(C.sp_get_port_name(p.p))
}

// Get a description for a port, to present to end user.
func (p *Port) Description() string {
	return C.GoString(C.sp_get_port_description(p.p))
}

// Get the transport type used by a port.
func (p *Port) Transport() int {
	t := C.sp_get_port_transport(p.p)
	return int(t)
}

// Get the USB bus number and address on bus of a USB serial adapter port.
func (p *Port) USBBusAddress() (int, int, error) {
	var bus, address C.int
	if err := errmsg(C.sp_get_port_usb_bus_address(p.p, &bus, &address)); err != nil {
		return 0, 0, err
	}
	return int(bus), int(address), nil
}

// Get the USB Vendor ID and Product ID of a USB serial adapter port.
func (p *Port) USBVIDPID() (int, int, error) {
	var vid, pid C.int
	if err := errmsg(C.sp_get_port_usb_vid_pid(p.p, &vid, &pid)); err != nil {
		return 0, 0, err
	}
	return int(vid), int(pid), nil
}

// Get the USB manufacturer string of a USB serial adapter port.
func (p *Port) USBManufacturer() string {
	cdesc := C.sp_get_port_usb_manufacturer(p.p)
	return C.GoString(cdesc)
}

// Get the USB product string of a USB serial adapter port.
func (p *Port) USBProduct() string {
	cdesc := C.sp_get_port_usb_product(p.p)
	return C.GoString(cdesc)
}

// Get the USB serial number string of a USB serial adapter port.
func (p *Port) USBSerialNumber() string {
	cdesc := C.sp_get_port_usb_serial(p.p)
	return C.GoString(cdesc)
}

// Get the MAC address of a Bluetooth serial adapter port.
func (p *Port) BluetoothAddress() string {
	cdesc := C.sp_get_port_bluetooth_address(p.p)
	return C.GoString(cdesc)
}

// Open a port by name with Options.
func Open(name string, opt Options) (*Port, error) {
	p, err := PortByName(name)
	if err != nil {
		return nil, err
	}
	err = p.open(&opt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Open the serial port with the given options.
func (p *Port) Open(opt Options) error {
	return p.open(&opt)
}

func (p *Port) open(opt *Options) (err error) {
	// set default mode to read/write
	mode := MODE_READ | MODE_WRITE
	if opt.Mode != 0 {
		mode = opt.Mode
	}
	// open port with mode
	if err = errmsg(C.sp_open(p.p, C.enum_sp_mode(mode))); err != nil {
		return
	}

	// get port config
	if err = errmsg(C.sp_get_config(p.p, p.c)); err != nil {
		p.Close()
		return
	}

	// set baudrate
	if opt.Baudrate == 0 {
		opt.Baudrate = 9600
	} else {
		p.SetBaudrate(opt.Baudrate)
	}

	// set data bits
	if opt.DataBits == 0 {
		p.SetDataBits(8)
	} else {
		p.SetDataBits(opt.DataBits)
	}

	// set stop bits
	if opt.StopBits == 0 {
		p.SetStopBits(1)
	} else {
		p.SetStopBits(opt.StopBits)
	}

	// set parity
	p.SetParity(opt.Parity)

	// set flow control
	p.SetRTS(RTS_OFF)
	p.SetDTR(DTR_OFF)
	p.SetFlowControl(opt.FlowControl)

	// apply config
	p.ApplyConfig()

	// get port handle
	if err = errmsg(C.sp_get_port_handle(p.p, unsafe.Pointer(&p.fd))); err != nil {
		p.Close()
		return
	}
	// open file
	p.f = os.NewFile(p.fd, p.Name())
	p.opened = true

	return nil
}

// Close the serial port.
func (p *Port) Close() error {
	err := errmsg(C.sp_close(p.p))
	p.opened = false
	return err
}

// Get the baud rate from a port configuration. The port must be
// opened for this operation.
func (p *Port) Baudrate() (int, error) {
	var baudrate C.int
	if err := errmsg(C.sp_get_config_baudrate(p.c, &baudrate)); err != nil {
		return 0, err
	}
	return int(baudrate), nil
}

// Set the baud rate for the serial port. The port must be opened for
// this operation. Call p.ApplyConfig() to apply the change.
func (p *Port) SetBaudrate(baudrate int) error {
	return errmsg(C.sp_set_baudrate(p.p, C.int(baudrate)))
}

// Get the data bits from a port configuration. The port must be
// opened for this operation.
func (p *Port) DataBits() (int, error) {
	var bits C.int
	if err := errmsg(C.sp_get_config_bits(p.c, &bits)); err != nil {
		return 0, err
	}
	return int(bits), nil
}

// Set the number of data bits for the serial port. The port must be
// opened for this operation. Call p.ApplyConfig() to apply the
// change.
func (p *Port) SetDataBits(bits int) error {
	return errmsg(C.sp_set_config_bits(p.c, C.int(bits)))
}

// Get the parity setting from a port configuration. The port must be
// opened for this operation.
func (p *Port) Parity() (int, error) {
	var parity C.enum_sp_return
	if err := errmsg(C.sp_get_config_parity(p.c, &parity)); err != nil {
		return 0, err
	}
	return int(parity), nil
}

// Set the parity setting for the serial port. The port must be opened
// for this operation. Call p.ApplyConfig() to apply the change.
func (p *Port) SetParity(parity int) error {
	return errmsg(C.sp_set_config_parity(p.c, C.enum_sp_return(parity)))
}

// Get the stop bits from a port configuration. The port must be
// opened for this operation.
func (p *Port) StopBits() (int, error) {
	var stopbits C.int
	if err := errmsg(C.sp_get_config_stopbits(p.c, &stopbits)); err != nil {
		return 0, err
	}
	return int(stopbits), nil
}

// Set the stop bits for the serial port. The port must be opened for
// this operation. Call p.ApplyConfig() to apply the change.
func (p *Port) SetStopBits(stopbits int) error {
	return errmsg(C.sp_set_config_stopbits(p.c, C.int(stopbits)))
}

// Get the RTS pin behaviour from a port configuration. The port must
// be opened for this operation.
func (p *Port) RTS() (int, error) {
	var rts C.enum_sp_rts
	if err := errmsg(C.sp_get_config_rts(p.c, &rts)); err != nil {
		return 0, err
	}
	return int(rts), nil
}

// Set the RTS pin behaviour in a port configuration. The port must be
// opened for this operation. Call p.ApplyConfig() to apply the
// change.
func (p *Port) SetRTS(rts int) error {
	return errmsg(C.sp_set_config_rts(p.c, C.enum_sp_rts(rts)))
}

// Get the CTS pin behaviour from a port configuration. The port must
// be opened for this operation.
func (p *Port) CTS() (int, error) {
	var cts C.enum_sp_cts
	if err := errmsg(C.sp_get_config_cts(p.c, &cts)); err != nil {
		return 0, err
	}
	return int(cts), nil
}

// Set the CTS pin behaviour in a port configuration. The port must be
// opened for this operation. Call p.ApplyConfig() to apply the
// change.
func (p *Port) SetCTS(cts int) error {
	return errmsg(C.sp_set_config_cts(p.c, C.enum_sp_cts(cts)))
}

// Get the DTR pin behaviour from a port configuration. The port must
// be opened for this operation.
func (p *Port) DTR() (int, error) {
	var dtr C.enum_sp_rts
	if err := errmsg(C.sp_get_config_dtr(p.c, &dtr)); err != nil {
		return 0, err
	}
	return int(dtr), nil
}

// Set the DTR pin behaviour in a port configuration. The port must be
// opened for this operation. Call p.ApplyConfig() to apply the
// change.
func (p *Port) SetDTR(dtr int) error {
	return errmsg(C.sp_set_config_dtr(p.c, C.enum_sp_dtr(dtr)))
}

// Get the DSR pin behaviour from a port configuration. The port must
// be opened for this operation.
func (p *Port) DSR() (int, error) {
	var dsr C.enum_sp_dsr
	if err := errmsg(C.sp_get_config_dsr(p.c, &dsr)); err != nil {
		return 0, err
	}
	return int(dsr), nil
}

// Set the DSR pin behaviour in a port configuration. The port must be
// opened for this operation. Call p.ApplyConfig() to apply the
// change.
func (p *Port) SetDSR(dsr int) error {
	return errmsg(C.sp_set_config_dsr(p.c, C.enum_sp_dsr(dsr)))
}

// Get the XON/XOFF configuration from a port configuration. The port
// must be opened for this operation.
func (p *Port) XonXoff() (int, error) {
	var xon C.enum_sp_xonxoff
	if err := errmsg(C.sp_get_config_xon_xoff(p.c, &xon)); err != nil {
		return 0, err
	}
	return int(xon), nil
}

// Set the XON/XOFF configuration in a port configuration. The port
// must be opened for this operation. Call p.ApplyConfig() to apply
// the change.
func (p *Port) SetXonXoff(xon int) error {
	return errmsg(C.sp_set_config_xon_xoff(p.c, C.enum_sp_xonxoff(xon)))
}

// Set the flow control type in a port configuration. The port must be
// opened for this operation. Call p.ApplyConfig() to apply the
// change.
func (p *Port) SetFlowControl(xon int) error {
	return errmsg(C.sp_set_config_flowcontrol(p.c, C.enum_sp_flowcontrol(xon)))
}

// Apply the configuration for the serial port.
func (p *Port) ApplyConfig() error {
	return errmsg(C.sp_set_config(p.p, p.c))
}

// Apply the raw mode configuration for the serial port.
func (p *Port) ApplyRawConfig() (err error) {
	if err = p.SetDataBits(8); err != nil {
		return
	}
	if err = p.SetParity(PARITY_NONE); err != nil {
		return
	}
	if err = p.SetStopBits(1); err != nil {
		return
	}
	if err = p.SetFlowControl(FLOWCONTROL_NONE); err != nil {
		return
	}
	return errmsg(C.sp_set_config(p.p, p.c))
}

// Implementation of io.Reader interface.
func (p *Port) Read(b []byte) (int, error) {
	// use native read for no deadline
	if p.readDeadline.IsZero() {
		return p.f.Read(b)
	}

	// calculate milliseconds until deadline
	delta := p.readDeadline.Sub(time.Now())
	millis := delta.Nanoseconds() / int64(time.Millisecond)

	var c int32

	if millis <= 0 {
		// call nonblocking read
		c = C.sp_nonblocking_read(
			p.p, unsafe.Pointer(&b[0]), C.size_t(len(b)))
	} else {
		// call blocking read
		c = C.sp_blocking_read(
			p.p, unsafe.Pointer(&b[0]), C.size_t(len(b)), C.uint(millis))
	}

	// check for error
	if c < 0 {
		return 0, errmsg(c)
	}

	// update slice length
	reflect.ValueOf(&b).Elem().SetLen(int(c))

	return int(c), nil
}

// Implementation of io.Writer interface.
func (p *Port) Write(b []byte) (int, error) {
	if p.writeDeadline.IsZero() {
		return p.f.Write(b)
	}

	// calculate milliseconds until deadline
	delta := p.writeDeadline.Sub(time.Now())
	millis := delta.Nanoseconds() / int64(time.Millisecond)

	var c int32

	if millis <= 0 {
		// call nonblocking write
		c = C.sp_nonblocking_write(
			p.p, unsafe.Pointer(&b[0]), C.size_t(len(b)))
	} else {
		// call blocking write
		c = C.sp_blocking_write(
			p.p, unsafe.Pointer(&b[0]), C.size_t(len(b)), C.uint(millis))
	}

	// check for error
	if c < 0 {
		return 0, errmsg(c)
	}

	return int(c), nil
}

// WriteString is like Write, but writes the contents of string s
// rather than a slice of bytes.
func (p *Port) WriteString(s string) (int, error) {
	return p.f.WriteString(s)
}

// Implementation of net.Conn.LocalAddr
func (p *Port) LocalAddr() *Addr {
	return &Addr{name: p.Name()}
}

// Implementation of net.Conn.RemoteAddr
func (p *Port) RemoteAddr() *Addr {
	return &Addr{name: p.Name()}
}

// Implementation of net.Conn.SetDeadline
func (p *Port) SetDeadline(t time.Time) error {
	p.readDeadline = t
	p.writeDeadline = t
	return nil
}

// Implementation of net.Conn.SetReadDeadline
func (p *Port) SetReadDeadline(t time.Time) error {
	p.readDeadline = t
	return nil
}

// Implementation of net.Conn.SetWriteDeadline
func (p *Port) SetWriteDeadline(t time.Time) error {
	p.writeDeadline = t
	return nil
}

// Gets the number of bytes waiting in the input buffer.
func (p *Port) InputWaiting() (int, error) {
	c := C.sp_input_waiting(p.p)
	if c < 0 {
		return 0, errmsg(c)
	}
	return int(c), nil
}

// Gets the number of bytes waiting in the output buffer.
func (p *Port) OutputWaiting() (int, error) {
	c := C.sp_output_waiting(p.p)
	if c < 0 {
		return 0, errmsg(c)
	}
	return int(c), nil
}

// Alias for Flush.
func (p *Port) Sync() error {
	return p.Flush()
}

// Flush serial port buffers.
func (p *Port) Flush() error {
	return errmsg(C.sp_flush(p.p, C.SP_BUF_BOTH))
}

// Flush serial port input buffers.
func (p *Port) FlushInput() error {
	return errmsg(C.sp_flush(p.p, C.SP_BUF_INPUT))
}

// Flush serial port output buffers.
func (p *Port) FlushOutput() error {
	return errmsg(C.sp_flush(p.p, C.SP_BUF_OUTPUT))
}

// Wait for buffered data to be transmitted.
func (p *Port) Drain() error {
	return errmsg(C.sp_drain(p.p))
}

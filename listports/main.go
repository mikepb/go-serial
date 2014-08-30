package main

import (
	".."
	"log"
)

func main() {
	ports, err := serial.ListPorts()
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Found %d ports:\n", len(ports))

	for _, port := range ports {
		log.Println(port.Name())
		log.Println("\tName:", port.Name())
		log.Println("\tDescription:", port.Description())
		log.Println("\tTransport:", port.Transport())

		if bus, addr, err := port.USBBusAddress(); err != nil {
			log.Println("\tbus:", bus, "\taddr:", addr)
		} else {
			log.Println(err)
		}

		if vid, pid, err := port.USBVIDPID(); err != nil {
			log.Println("\tvid:", vid, "\tpid:", pid)
		} else {
			log.Println(err)
		}

		log.Println("\tUSB Manufacturer:", port.USBManufacturer())
		log.Println("\tUSB Product:", port.USBProduct())
		log.Println("\tUSB Serial Number:", port.USBSerialNumber())
		log.Println("\tBluetooth Address:", port.BluetoothAddress())

		if err := port.Open(serial.Options{}); err != nil {
			log.Println("\tOpen:", err)
			continue
		}

		if baudrate, err := port.Baudrate(); err != nil {
			log.Println("\tBaudrate:", err)
		} else {
			log.Println("\tBaudrate:", baudrate)
		}

		if databits, err := port.DataBits(); err != nil {
			log.Println("\tData Bits:", err)
		} else {
			log.Println("\tData Bits:", databits)
		}

		if parity, err := port.Parity(); err != nil {
			log.Println("\tParity:", err)
		} else {
			log.Println("\tParity:", parity)
		}

		if stopbits, err := port.StopBits(); err != nil {
			log.Println("\tStop Bits:", err)
		} else {
			log.Println("\tStop Bits:", stopbits)
		}

		if rts, err := port.RTS(); err != nil {
			log.Println("\tRTS:", err)
		} else {
			log.Println("\tRTS:", rts)
		}

		if cts, err := port.CTS(); err != nil {
			log.Println("\tCTS:", err)
		} else {
			log.Println("\tCTS:", cts)
		}

		if dtr, err := port.DTR(); err != nil {
			log.Println("\tDTR:", err)
		} else {
			log.Println("\tDTR:", dtr)
		}

		if dsr, err := port.DSR(); err != nil {
			log.Println("\tDSR:", err)
		} else {
			log.Println("\tDSR:", dsr)
		}

		if xon, err := port.XonXoff(); err != nil {
			log.Println("\tXON/XOFF:", err)
		} else {
			log.Println("\tXON/XOFF:", xon)
		}

		if err := port.ApplyRawConfig(); err != nil {
			log.Println("\tApply Raw Config:", err)
		} else {
			log.Println("\tApply Raw Config: ok")
		}

		buf := make([]byte, 1)
		if c, err := port.Read(buf); err != nil {
			log.Println("\tRead:", err)
		} else {
			log.Printf("\tRead %d: %v", c, buf)
		}

		if c, err := port.Write([]byte{0}); err != nil {
			log.Println("\tWrite:", err)
		} else {
			log.Printf("\tWrite %d: %v", c, buf)
		}

		if err := port.Close(); err != nil {
			log.Println(err)
			continue
		}
	}
}

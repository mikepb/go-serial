package main

import (
	".."
	"log"
	"time"
)

func main() {
	ports, err := serial.ListPorts()
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Found %d ports:\n", len(ports))

	for _, info := range ports {
		log.Println(info.Name())
		log.Println("\tName:", info.Name())
		log.Println("\tDescription:", info.Description())
		log.Println("\tTransport:", info.Transport())

		if bus, addr, err := info.USBBusAddress(); err != nil {
			log.Println("\tbus:", bus, "\taddr:", addr)
		} else {
			log.Println(err)
		}

		if vid, pid, err := info.USBVIDPID(); err != nil {
			log.Println("\tvid:", vid, "\tpid:", pid)
		} else {
			log.Println(err)
		}

		log.Println("\tUSB Manufacturer:", info.USBManufacturer())
		log.Println("\tUSB Product:", info.USBProduct())
		log.Println("\tUSB Serial Number:", info.USBSerialNumber())
		log.Println("\tBluetooth Address:", info.BluetoothAddress())

		port, err := info.Open()
		if err != nil {
			log.Println("\tOpen:", err)
			continue
		}

		log.Println("\tLocalAddr:", port.LocalAddr().String())
		log.Println("\tRemoteAddr:", port.RemoteAddr().String())

		if bitrate, err := port.BitRate(); err != nil {
			log.Println("\tBit Rate:", err)
		} else {
			log.Println("\tBit Rate:", bitrate)
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

		/*
			if err := port.Apply(&serial.RawOptions); err != nil {
				log.Println("\tApply Raw Config:", err)
			} else {
				log.Println("\tApply Raw Config: ok")
			}
		*/

		if b, err := port.InputWaiting(); err != nil {
			log.Println("\tInput Waiting: ", err)
		} else {
			log.Println("\tInput Waiting: ", b)
		}

		if b, err := port.OutputWaiting(); err != nil {
			log.Println("\tOutput Waiting: ", err)
		} else {
			log.Println("\tOutput Waiting: ", b)
		}

		if err := port.Sync(); err != nil {
			log.Println("\tSync: ", err)
		}
		if err := port.Reset(); err != nil {
			log.Println("\tReset: ", err)
		}
		if err := port.ResetInput(); err != nil {
			log.Println("\tReset input: ", err)
		}
		if err := port.ResetOutput(); err != nil {
			log.Println("\tReset output: ", err)
		}

		buf := make([]byte, 1)

		if err := port.SetDeadline(time.Now()); err != nil {
			log.Println("\tSetDeadline: ", err)
		} else {
			log.Printf("\tSet deadline")
		}

		if c, err := port.Read(buf); err != nil {
			log.Printf("\tRead immediate %d: %v", c, err)
			if err != serial.ErrTimeout {
				continue
			}
		} else {
			log.Printf("\tRead immediate %d: %v", c, buf)
		}

		if c, err := port.Write([]byte{0}); err != nil {
			log.Println("\tWrite immediate:", err)
			if err != serial.ErrTimeout {
				continue
			}
		} else {
			log.Printf("\tWrite immediate %d: %v", c, buf)
		}

		if err := port.SetDeadline(time.Now().Add(time.Millisecond)); err != nil {
			log.Println("\tSetDeadline: ", err)
		} else {
			log.Printf("\tSet deadline")
		}

		if c, err := port.Read(buf); err != nil {
			log.Printf("\tRead wait %d: %v", c, err)
		} else {
			log.Printf("\tRead wait %d: %v", c, buf)
		}

		if err := port.SetDeadline(time.Now().Add(time.Millisecond)); err != nil {
			log.Println("\tSetDeadline: ", err)
		} else {
			log.Printf("\tSet deadline")
		}

		if c, err := port.Write([]byte{0}); err != nil {
			log.Println("\tWrite wait:", err)
		} else {
			log.Printf("\tWrite wait %d: %v", c, buf)
		}

		if err := port.SetReadDeadline(time.Time{}); err != nil {
			log.Println("\tSetReadDeadline: ", err)
		} else {
			log.Printf("\tSet read deadline")
		}
		if err := port.SetWriteDeadline(time.Time{}); err != nil {
			log.Println("\tSetWriteDeadline: ", err)
		} else {
			log.Printf("\tSet write deadline")
		}

		if c, err := port.Read(buf); err != nil {
			log.Printf("\tRead %d: %v", c, err)
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
		}
	}
}

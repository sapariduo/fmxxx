package handlers

import (
	"bytes"
	"fmt"

	"github.com/leesper/holmes"
)

func parseData(data []byte, size int, imei string) (elements []Record, err error) {
	reader := bytes.NewBuffer(data)
	// fmt.Println("Reader Size:", reader.Len())
	// Header
	reader.Next(4)                                    // 4 Zero Bytes
	dataLength, err := streamToInt32(reader.Next(4))  // Header
	reader.Next(1)                                    // CodecID
	recordNumber, err := streamToInt8(reader.Next(1)) // Number of Records
	holmes.Debugf("Length of data: %d \n", dataLength)

	elements = make([]Record, recordNumber)

	var i int8 = 0
	for i < recordNumber {
		timestamp, err := streamToTime(reader.Next(8)) // Timestamp
		reader.Next(1)                                 // Priority

		// GPS Element
		longitudeInt, err := streamToInt32(reader.Next(4)) // Longitude
		longitude := float64(longitudeInt) / PRECISION
		latitudeInt, err := streamToInt32(reader.Next(4)) // Latitude
		latitude := float64(latitudeInt) / PRECISION

		altitude, err := streamToInt16(reader.Next(2)) // Altitude
		angle, err := streamToInt16(reader.Next(2))    // Angle
		sats, err := streamToInt8(reader.Next(1))      // Satellites
		speed, err := streamToInt16(reader.Next(2))    // Speed
		evtID, err := streamToUInt8(reader.Next(1))    // ioEventID

		if err != nil {
			holmes.Errorln("Error while reading GPS Element")
			break
		}
		io := make(map[string]interface{})

		elements[i] = Record{
			imei,
			Location{"Point",
				[]float64{longitude, latitude}},
			timestamp,
			angle,
			speed,
			altitude,
			sats,
			evtID,
			io,
		}

		// IO Events Elements

		// reader.Next(1) // ioEventID
		reader.Next(1) // total Elements

		stage := 1
		for stage <= 4 {
			stageElements, err := streamToInt8(reader.Next(1))
			if err != nil {
				break
			}

			var j int8 = 0
			for j < stageElements {
				_x, _ := streamToUInt8(reader.Next(1)) // elementID
				x := fmt.Sprintf("%d", _x)
				var val interface{}
				switch stage {
				case 1: // One byte IO Elements
					val, err = streamToInt8(reader.Next(1))
					io[x] = val
				case 2: // Two byte IO Elements
					val, err = streamToInt16(reader.Next(2))
					io[x] = val
				case 3: // Four byte IO Elements
					val, err = streamToInt32(reader.Next(4))
					io[x] = val
				case 4: // Eigth byte IO Elements
					val, err = streamToInt64(reader.Next(8))
					io[x] = val
				}
				j++
			}
			stage++
		}

		if err != nil {
			fmt.Println("Error while reading IO Elements")
			break
		}

		holmes.Debugln("Timestamp:", timestamp)
		holmes.Debugln("Longitude:", longitude, "Latitude:", latitude)

		i++
	}

	// Once finished with the records we read the Record Number and the CRC

	_, err = streamToInt8(reader.Next(1))  // Number of Records
	_, err = streamToInt32(reader.Next(4)) // CRC

	return
}

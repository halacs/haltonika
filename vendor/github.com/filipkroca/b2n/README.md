# Package b2n provides functions for parsing and conversion values from byte arrays and slices

Certain purpose:

Package b2n was created for parsing data from [Teltonika](https://wiki.teltonika.lt/view/Codec#Codec_8_Extended) UDP packets, package can be used for parsing values from data streams.

For example, have binary packet bs which is Teltonika UDP Codec 8 Extended packet according to the Teltonika Codec 8 documentation, on byte position 24 should be number of data which should be unsigned uint8, use ParseBs2Uint8 function to parse and convert the value.

```go
package main

import (
    "encoding/hex"
    "fmt"

    "github.com/filipkroca/b2n"
)

func main() {
    // create HEX string raw
    var dataString = "00A1CAFE001B000F3335363330373034323434313031338E010000013FEBDD19C8000F0E9FF0209A718000690000120000001E09010002000300040016014703F0001504C8000C0900910A00440B004D130044431555440000B5000BB60005422E9B180000CD0386CE000107C700000000F10000601A460000013C4800000BB84900000BB84A00000BB84C00000000024E0000000000000000CF000000000000000001"

    // decode sting into a byte slice
    bs, _ := hex.DecodeString(dataString)

    // parse a value on Byte offset 24, is should be number of data according to the Teltonika documentation
    noOfData, err := b2n.ParseBs2Uint8(&bs, 24)
    if err != nil {
        fmt.Printf(err)
        return
    }

    fmt.Println("%T %v", noOfData)
}
```

Output should be

```text
uint8 1
```

Full documentation [HERE](https://godoc.org/github.com/filipkroca/b2n#example-ValidateIMEI)

## Bytes slice to unsigned integer

### ParseBs2Uint8  

ParseBs2Uint8 takes a pointer to a byte slice, start byte and returns parsed Uint8 and error

Performance per core:   0.46 ns/op, 0 B/op, 0 allocs/op

[DOCUMENTATION](https://godoc.org/github.com/filipkroca/b2n#ParseBs2Uint8)
[EXAMPLE](https://godoc.org/github.com/filipkroca/b2n#example-ParseBs2Uint8)

### ParseBs2Uint16  

ParseBs2Uint16 takes a pointer to a byte slice, start byte and returns parsed Uint16 and error

Performance per core:   3.35 ns/op, 0 B/op, 0 allocs/op

[DOCUMENTATION](https://godoc.org/github.com/filipkroca/b2n#ParseBs2Uint16)
[EXAMPLE](https://godoc.org/github.com/filipkroca/b2n#example-ParseBs2Uint16)

### ParseBs2Uint32  

ParseBs2Uint32 takes a pointer to a byte slice, start byte and returns parsed Uint32 and error

Performance per core:   4.97 ns/op, 0 B/op, 0 allocs/op

[DOCUMENTATION](https://godoc.org/github.com/filipkroca/b2n#ParseBs2Uint32)
[EXAMPLE](https://godoc.org/github.com/filipkroca/b2n#example-ParseBs2Uint32)

## Bytes slice encoded with Two's complement to signed integer  

Read more here [Two's complement](https://en.wikipedia.org/wiki/Two%27s_complement)  

### ParseBs2Int8TwoComplement  

ParseBs2Int8TwoComplement takes a pointer to a byte slice coded with Two Complement Arithmetic, start byte and returns parsed signed Int8 and error

Performance per core:   0.24 ns/op, 0 B/op, 0 allocs/op

[DOCUMENTATION](https://godoc.org/github.com/filipkroca/b2n#ParseBs2Int8TwoComplement)
[EXAMPLE](https://godoc.org/github.com/filipkroca/b2n#example-ParseBs2Int8TwoComplement)

### ParseBs2Int16TwoComplement  

ParseBs2Int16TwoComplement takes a pointer to a byte slice coded with Two Complement Arithmetic, start byte and returns parsed signed Int16 and error

Performance per core:   4.52 ns/op, 0 B/op, 0 allocs/op

[DOCUMENTATION](https://godoc.org/github.com/filipkroca/b2n#ParseBs2Int16TwoComplement)
[EXAMPLE](https://godoc.org/github.com/filipkroca/b2n#example-ParseBs2Int16TwoComplement)

### ParseBs2Int32TwoComplement  

ParseBs2Int32TwoComplement takes a pointer to a byte slice coded with Two Complement Arithmetic, start byte and returns parsed signed Int32 and error

Performance per core:   7.48 ns/op, 0 B/op, 0 allocs/op

[DOCUMENTATION](https://godoc.org/github.com/filipkroca/b2n#ParseBs2Int32TwoComplement)
[EXAMPLE](https://godoc.org/github.com/filipkroca/b2n#example-ParseBs2Int32TwoComplement)  

### ParseBs2Int64TwoComplement  

ParseBs2Int64TwoComplement takes a pointer to a byte slice coded with Two Complement Arithmetic, start byte and returns parsed signed Int64 and error

Performance per core:   11.1 ns/op, 0 B/op, 0 allocs/op

[DOCUMENTATION](https://godoc.org/github.com/filipkroca/b2n#ParseBs2Int64TwoComplement)
[EXAMPLE](https://godoc.org/github.com/filipkroca/b2n#example-ParseBs2Int64TwoComplement)  

## IMEI number functions

This functions provide support for parsing and validating [International Mobile Equipment Identity](https://en.wikipedia.org/wiki/International_Mobile_Equipment_Identity)

For example, have binary packet bs which is Teltonika UDP Codec 8 packet

```go
package main

import (
    "encoding/hex"
    "fmt"

    "github.com/filipkroca/b2n"
)

func main() {
    //Example packet Teltonika UDP Codec 8 007CCAFE0133000F33353230393430383136373231373908020000016C32B488A0000A7A367C1D30018700000000000000F1070301001500EF000342318BCD42DCCE606401F1000059D9000000016C32B48C88000A7A367C1D3001870000000000000015070301001501EF0003423195CD42DCCE606401F1000059D90002, IMEI is located starting byte 8

    // create a raw byte slice
    var bs = []byte{0x00, 0x7C, 0xCA, 0xFE, 0x01, 0x33, 0x00, 0x0F, 0x33, 0x35, 0x32, 0x30, 0x39, 0x34, 0x30, 0x38, 0x31, 0x36, 0x37, 0x32, 0x31, 0x37, 0x39, 0x08}

    //parse and validate imei
    imei, err := ParseIMEI(&bs, 8, 15)
    if err != nil {
        fmt.Println("ExampleParseIMEI error", err)
    }
    fmt.Printf("%T %v", imei, imei)

}
```

According to the Teltonika Codec 8 documentation, on byte position 8 should be 15 digits long IMEI number. After parsing and validating

Output should be

```text
string 352094081672179
```

### ParseIMEI

ParseIMEI takes a pointer to a byte slice including IMEI number encoded as ASCII, IMEI length, offset and returns IMEI as string and error. If len is 15 digits, also do imei validation

Performance per core 235 ns/op, 16 B/op, 1 allocs/op

[DOCUMENTATION](https://godoc.org/github.com/filipkroca/b2n#ParseIMEI)
[EXAMPLE](https://godoc.org/github.com/filipkroca/b2n#example-ParseIMEI)

### ValidateIMEI

ValidateIMEI takes pointer to 15 digits long IMEI string, calculate checksum using the Luhn algorithm and return validity as bool

Performance per core 193 ns/op, 0 B/op, 0 allocs/op

[DOCUMENTATION](https://godoc.org/github.com/filipkroca/b2n#ValidateIMEI)
[EXAMPLE](https://godoc.org/github.com/filipkroca/b2n#example-ValidateIMEI)

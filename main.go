package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

func fatal(a ...any) {
	fmt.Println(a...)
	os.Exit(1)
}

func main() {
	flag.Parse()
	args := flag.Args()
	filename := args[0]

	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		fatal(err)
	}
	if len(bytes)%512 != 0 {
		fatal("size is not 512-byte-aligned")
	}
	rb := newReadBuffer(bytes)
	out := readChunk(rb)
	for out != nil {
		fmt.Print(out.Header())
		fmt.Println()
		fmt.Print(out.HexPayload())
		fmt.Println()
		fmt.Print(out.DisPayload())
		out = readChunk(rb)
	}
}

func hexPrint(payload []byte) string {
	out := "\t"
	for i, b := range payload {
		if i != 0 && i%2 == 0 {
			out += " "
		}
		if i != 0 && i%16 == 0 {
			out += "\n\t"
		}
		out += fmt.Sprintf("%02X", b)
	}
	out += "\n"
	return out
}

type uf2block struct {
	flags       uf2flags
	addr        uint32
	payloadSize uint32
	seqBlockNum uint32
	totBlockNum uint32

	something uint32

	payload []byte
}

func (this uf2block) Header() string {
	out := ""
	out += fmt.Sprintf("\tflags:\t%v\n", this.flags)
	out += fmt.Sprintf("\twaddr:\t0x%08X\n", this.addr)
	out += fmt.Sprintf("\tsize:\t%v\n", this.payloadSize)
	out += fmt.Sprintf("\tseqnum:\t%v\n", this.seqBlockNum)
	out += fmt.Sprintf("\ttotnum:\t%v\n", this.totBlockNum)
	if this.flags.FamilyIDPresent {
		out += fmt.Sprintf("\tfamID:\t0x%08X\n", this.something)
	} else {
		out += fmt.Sprintf("\tsome:\t0x%08X\n", this.something)
	}
	return out
}

func (this uf2block) HexPayload() string {
	return hexPrint(this.payload)
}

func (this uf2block) DisPayload() string {
	out := ""
	rb := newReadBuffer(this.payload)

	var instrOut instr
	startAddr := this.addr
	for decodeInstr(rb, &instrOut) {
		out += fmt.Sprintf("%08X", startAddr) +
			" " + strchunk(instrOut.chunk) +
			"\t" + instrOut.text + "\n"
		startAddr += instrOut.size
	}

	return out
}

func strchunk(chunk []byte) string {
	if len(chunk) == 2 {
		return fmt.Sprintf("    %02X%02X", chunk[1], chunk[0])
	} else if len(chunk) == 4 {
		return fmt.Sprintf("%02X%02X%02X%02X", chunk[3], chunk[2], chunk[1], chunk[0])
	} else {
		panic("wtf")
	}
}

type instr struct {
	text  string
	size  uint32
	chunk []byte
}

const (
	bits15_12 uint16 = 0b1111_0000_0000_0000
	bits15_11 uint16 = 0b1111_1000_0000_0000
	bits15_10 uint16 = 0b1111_1100_0000_0000
	bits15_9  uint16 = 0b1111_1110_0000_0000
	bits15_8  uint16 = 0b1111_1111_0000_0000
	bits15_7  uint16 = 0b1111_1111_1000_0000
	bits15_6  uint16 = 0b1111_1111_1100_0000

	bits10_0 uint16 = 0b0000_0111_1111_1111
	bits7_0  uint16 = 0b0000_0000_1111_1111
	bits6_0  uint16 = 0b0000_0000_0111_1111

	bits6_3 uint16 = 0b0000_0000_0111_1000

	bits11_8 uint16 = 0b0000_1111_0000_0000
	bits10_8 uint16 = 0b0000_0111_0000_0000
	bits10_6 uint16 = 0b0000_0111_1100_0000
	bits8_6  uint16 = 0b0000_0001_1100_0000
	bits5_3  uint16 = 0b0000_0000_0011_1000
	bits2_0  uint16 = 0b0000_0000_0000_0111

	bit7 uint16 = 0b0000_0000_1000_0000
)

func decodeInstr(rb *ReadBuffer, out *instr) bool {
	hw, ok := rb.getU16()
	if !ok {
		return false
	}
	first := uint8((hw >> 8) & 0xFF)
	last := uint8(hw & 0xFF)

	out.chunk = []byte{last, first} // little endian
	out.text = "???"
	out.size = 2

	if hw&bits15_6 == 0b0100_0001_0100_0000 { // ADCS <Rdn>, <Rm>
		rm := (hw & bits5_3) >> 3
		rdn := hw & bits2_0
		out.text = fmt.Sprintf("ADCS %v, %v", reg(rdn), reg(rm))
	}

	if hw&bits15_9 == 0b0001_1100_0000_0000 { // ADDS <Rd>, <Rn>, #<imm3>
		imm3 := (hw & bits8_6) >> 6
		rn := (hw & bits5_3) >> 3
		rd := hw & bits2_0
		out.text = fmt.Sprintf("ADDS %v, %v, #%v", reg(rd), reg(rn), imm3)
	}

	if hw&bits15_11 == 0b0011_0000_0000_0000 { // ADDS <Rdn>, #<imm8>
		imm8 := hw & bits7_0
		rdn := (hw & bits10_8) >> 8
		out.text = fmt.Sprintf("ADDS %v, #%v", reg(rdn), imm8)
	}

	if hw&bits15_9 == 0b0001_1000_0000_0000 { // ADDS <Rd>, <Rn>, <Rm>
		rm := (hw & bits8_6) >> 6
		rn := (hw & bits5_3) >> 3
		rd := (hw & bits2_0)
		out.text = fmt.Sprintf("ADDS %v, %v, %v", reg(rd), reg(rn), reg(rm))
	}

	if hw&bits15_8 == 0b0100_0100_0000_0000 { // ADD <Rdn>, <Rm>
		DN := (hw & bit7) >> 4
		rdn := hw & bits2_0
		d := DN | rdn
		rm := (hw & bits6_3) >> 3
		out.text = fmt.Sprintf("ADD %v, %v", reg(d), reg(rm))
	}

	if hw&bits15_11 == 0b1010_1000_0000_0000 { // ADD <Rd>, SP, #<imm8>
		rd := (hw & bits10_8) >> 8
		imm8 := (hw & bits7_0) << 2
		out.text = fmt.Sprintf("ADD %v, SP, %v", reg(rd), imm8)
	}

	if hw&bits15_7 == 0b1011_0000_0000_0000 { // ADD SP, SP, #<imm7>
		imm7 := (hw & bits6_0) << 2
		out.text = fmt.Sprintf("ADD SP, SP, %v", imm7)
	}

	if hw&(bits15_8|bits6_3) == 0b0100_0100_0110_1000 { // ADD <Rdm>, SP, <Rdm>
		DM := (hw & bit7) >> 4
		Rdm := (hw & bits2_0)
		d := DM | Rdm
		out.text = fmt.Sprintf("ADD %v, SP, %v", d, d)
	}

	if hw&(bits15_7|bits2_0) == 0b0100_0100_1000_0101 { // ADD SP, <rm>
		rm := (hw & bits6_3) >> 3
		out.text = fmt.Sprintf("ADD SP, %v", reg(rm))
	}

	if hw&bits15_11 == 0b1010_0000_0000_0000 { // ADR <Rd>, PC, #<const>
		rd := (hw & bits10_8) >> 8
		imm8 := (hw & bits7_0) << 2
		out.text = fmt.Sprintf("ADR %v, PC, #%v", reg(rd), imm8)
	}

	if hw&bits15_6 == 0b0100_0000_0000_0000 { // ANDS <Rdn>, <Rm>
		rdn := hw & bits2_0
		rm := (hw & bits5_3) >> 3
		out.text = fmt.Sprintf("ANDS %v, %v", reg(rdn), reg(rm))
	}

	if hw&bits15_11 == 0b0001_0000_0000_0000 { // ASRS <Rd>, <Rm>, #<imm5>
		rm := (hw & bits5_3) >> 3
		rd := hw & bits2_0
		imm5 := (hw & bits10_6) >> 6

		shift_n := imm5
		if imm5 == 0b00000 {
			shift_n = 32
		}
		out.text = fmt.Sprintf("ASRS %v, %v, #%v", reg(rd), reg(rm), shift_n)
	}

	if hw&bits15_6 == 0b0100_0001_0000_0000 { // ASRS <Rdn>, <Rm>
		rm := (hw & bits5_3) >> 3
		rdn := hw & bits2_0
		out.text = fmt.Sprintf("ASRS %v, %v", reg(rdn), reg(rm))
	}

	if hw&bits15_12 == 0b1101_0000_0000_0000 { // B<c> <label>
		c := uint8((hw & bits11_8) >> 8)
		imm8 := hw & bits7_0
		if c == 0b1110 {
			out.text = fmt.Sprintf("UDF #%v", imm8)
		} else if c == 0b1111 {
			out.text = fmt.Sprintf("SVC #%v", imm8)
		} else {
			offset := int32(int8(imm8)) << 1
			out.text = fmt.Sprintf("B%v <PC, %v>", cond(c), offset)
		}
	}

	if hw&bits15_11 == 0b1110_0000_0000_0000 { // b <label>
		imm11 := hw & bits10_0
		imm32 := int32(int16(imm11<<5) >> 4)
		out.text = fmt.Sprintf("B <PC, %v>", imm32)
	}

	if hw&bits15_6 == 0b0100_0011_1000_0000 { // BICS <rdn>, <rm>
		rm := (hw & bits5_3) >> 3
		rdn := hw & bits2_0
		out.text = fmt.Sprintf("BICS %v, %v", reg(rdn), reg(rm))
	}

	if hw&bits15_8 == 0b1011_1110_0000_0000 { // BKPT #<imm8>
		imm8 := hw & bits7_0
		out.text = fmt.Sprintf("BKPT #%v", imm8)
	}

	if hw&bits15_10 == 0b1111_0000_0000_0000 { // BL <label>
	}

	if hw&(bits15_7|bits2_0) == 0b0100_0111_1000_0000 { // BLX <rm>
		rm := (hw & bits6_3) >> 3
		out.text = fmt.Sprintf("BLX %v", reg(rm))
	}

	if hw&(bits15_7|bits2_0) == 0b0100_0111_0000_0000 { // BX <rm>
		rm := (hw & bits6_3) >> 3
		out.text = fmt.Sprintf("BX %v", reg(rm))
	}

	if hw&bits15_6 == 0b0100_0010_1100_0000 { // CMN <rn>, <rm>
		rm := (hw & bits5_3) >> 3
		rn := (hw & bits2_0)
		out.text = fmt.Sprintf("CMN %v, %v", reg(rn), reg(rm))
	}

	if hw&bits15_11 == 0b0010_1000_0000_0000 { // CMP <rn>, #<imm8>
		rn := (hw & bits10_8) >> 8
		imm8 := (hw & bits7_0)
		out.text = fmt.Sprintf("CMP %v, #%v", reg(rn), imm8)
	}

	if hw&bits15_6 == 0b0100_0010_1000_0000 { // CMP <rn>, <rm>
		rm := (hw & bits5_3) >> 3
		rn := hw & bits2_0
		out.text = fmt.Sprintf("CMP %v, %v", reg(rn), reg(rm))
	}

	if hw&bits15_8 == 0b0100_0101_0000_0000 { // CMP <rn>, <rm>
		N := (hw & bit7) >> 4
		rn := hw & bits2_0
		n := N | rn
		rm := (hw & bits6_3) >> 3
		out.text = fmt.Sprintf("CMP %v, %v", reg(n), reg(rm))
	}

	if hw == 0b1111_0011_1011_1111 { // DMB
	}

	if hw == 0b1111_0011_1011_1111 { // DSB
	}

	if hw&bits15_11 == 0b1100_1000_0000_0000 { // LDM <rn>, <registers>
		rn := (hw & bits10_8) >> 8
		list := (hw & bits7_0)
		out.text = fmt.Sprintf("LDM %v, %v", reg(rn), reglist(list))
	}

	if hw&bits15_6 == 0b0100_0011_0000_0000 { // ORR <rdn>, <rm>
		rdn := hw & bits2_0
		rm := (hw & bits5_3) >> 3
		out.text = fmt.Sprintf("ORRS %v, %v", reg(rdn), reg(rm))
	}

	if hw&bits15_6 == 0b1011_0010_0100_0000 { // SXTB <rd>, <rm>
		rm := hw & bits5_3 >> 3
		rd := hw & bits2_0
		out.text = fmt.Sprintf("SXTB %v, %v", reg(rd), reg(rm))
	}

	if hw&bits15_6 == 0b1011_0010_0000_0000 { // SXTH <rd>, <rm
		rm := hw & bits5_3 >> 3
		rd := hw & bits2_0
		out.text = fmt.Sprintf("SXTH %v, %v", reg(rd), reg(rm))
	}
	return true
}

func reglist(hw uint16) string {
	out := "{"
	var mask uint16 = 1
	var i uint16 = 0
	firsted := true
	for i = 0; i < 16; i++ {
		if hw&(mask<<i) > 0 {
			if !firsted {
				out += ", "
			}
			out += reg(i)
			firsted = false
		}
	}
	out += "}"
	return out
}

func cond(nibble uint8) string {
	cond := nibble & 0b0000_1111
	switch cond {
	case 0b0000:
		return "EQ"
	case 0b0001:
		return "NE"
	case 0b0010:
		return "CS"
	case 0b0011:
		return "CC"
	case 0b0100:
		return "MI"
	case 0b0101:
		return "PL"
	case 0b0110:
		return "VS"
	case 0b0111:
		return "VC"
	case 0b1000:
		return "HI"
	case 0b1001:
		return "LS"
	case 0b1010:
		return "GE"
	case 0b1011:
		return "LT"
	case 0b1100:
		return "GT"
	case 0b1101:
		return "LE"
	case 0b1110:
		return "?"
	default:
		return "!"
	}
}

func reg(r uint16) string {
	if r == 13 {
		return "sp"
	}
	if r == 14 {
		return "lr"
	}
	if r == 15 {
		return "pc"
	}
	return fmt.Sprintf("r%v", r)
}

func readChunk(rb *ReadBuffer) *uf2block {
	if rb.len() < 512 {
		return nil
	}
	magic1, _ := rb.getU32()
	if magic1 != 0x0A324655 {
		fmt.Printf("ERROR: first magic number is wrong, found: 0x%X, expected: 0x0A324655\n", magic1)
		return nil
	}
	magic2, _ := rb.getU32()
	if magic2 != 0x9E5D5157 {
		fmt.Printf("ERROR: second magic number is wrong, found: 0x%X, expected: 0x9E5D5157\n", magic2)
		return nil
	}
	block := uf2block{}

	// we can safely ignore the second return because we checked for
	// the size of the whole block
	f, _ := rb.getU32()
	block.flags = decodeFlags(f)
	block.addr, _ = rb.getU32()
	block.payloadSize, _ = rb.getU32()
	block.seqBlockNum, _ = rb.getU32()
	block.totBlockNum, _ = rb.getU32()
	block.something, _ = rb.getU32()

	block.payload = rb.data[rb.start : rb.start+int(block.payloadSize)]

	rb.start += 476

	magic3, _ := rb.getU32()
	if magic3 != 0x0AB16F30 {
		fmt.Printf("ERROR: final magic number is wrong, found: 0x%X, expected: 0x0AB16F30\n", magic3)
		return nil
	}

	return &block
}

type uf2flags struct {
	NotMainFlash         bool
	FileContainer        bool
	FamilyIDPresent      bool
	ChecksumPresent      bool
	ExtensionTagsPresent bool
}

func (this uf2flags) String() string {
	out := ""
	if this.NotMainFlash {
		out += "N"
	}
	if this.FileContainer {
		out += "F"
	}
	if this.FamilyIDPresent {
		out += "I"
	}
	if this.ChecksumPresent {
		out += "5"
	}
	if this.ExtensionTagsPresent {
		out += "X"
	}
	return out
}

func decodeFlags(flags uint32) uf2flags {
	out := uf2flags{}
	if flags&0x00000001 > 0 {
		out.NotMainFlash = true
	}
	if flags&0x00001000 > 0 {
		out.FileContainer = true
	}
	if flags&0x00002000 > 0 {
		out.FamilyIDPresent = true
	}
	if flags&0x00004000 > 0 {
		out.ChecksumPresent = true
	}
	if flags&0x00008000 > 0 {
		out.ExtensionTagsPresent = true
	}
	return out
}

type ReadBuffer struct {
	data  []byte
	start int
}

func newReadBuffer(data []byte) *ReadBuffer {
	return &ReadBuffer{
		data:  data,
		start: 0,
	}
}

func (this *ReadBuffer) len() int {
	return len(this.data) - this.start
}

// little endian
func (this *ReadBuffer) getU32() (uint32, bool) {
	if this.start+3 >= len(this.data) {
		return 0, false
	}
	out := uint32(this.data[this.start+3]) << 24
	out |= uint32(this.data[this.start+2]) << 16
	out |= uint32(this.data[this.start+1]) << 8
	out |= uint32(this.data[this.start])
	this.start += 4
	return out, true
}

// little endian
func (this *ReadBuffer) getU16() (uint16, bool) {
	if this.start+1 >= len(this.data) {
		return 0, false
	}
	out := uint16(this.data[this.start+1]) << 8
	out |= uint16(this.data[this.start])
	this.start += 2
	return out, true
}

func (this *ReadBuffer) getU8() (uint8, bool) {
	if this.start >= len(this.data) {
		return 0, false
	}
	out := this.data[this.start]
	this.start += 1
	return out, true
}

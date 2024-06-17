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

	if first&0b1111_1000 == 0b0010_0000 { // movs Rd, #imm8
		rd := first & 0b0000_0111
		out.text = fmt.Sprintf("movs %s, #%v", reg(rd), last)
	}

	if first == 0b0100_0110 { // mov Rd, Rm
		rm := (last & 0b0111_1000) >> 3
		rd := ((last & 0b1000_0000) >> 4) | (last & 0b0000_0111)
		out.text = fmt.Sprintf("mov %s, %s", reg(rd), reg(rm))
	}

	if first&0b1111_1000 == 0b0100_1000 { // ldr Rt, [PC, #imm8]
		rt := first & 0b0000_0111
		imm32 := uint32(last) << 2
		out.text = fmt.Sprintf("ldr %s, [pc, #%02X]", reg(rt), imm32)
	}

	if first == 0b1011_1111 && last == 0b0000_0000 { // nop
		out.text = "nop"
	}

	if first == 0b0100_0011 && last&0b1100_0000 == 0 { // ORR rdn, rm
		rm := (last & 0b0011_1000) >> 3
		rdn := (last & 0b00000_0111)
		out.text = fmt.Sprintf("orr %s, %s", reg(rdn), reg(rm))
	}

	if first&0b1111_1000 == 0b0110_0000 { // str rt, [rn, #imm5]
		rn := (last & 0b0011_1000) >> 3
		rt := (last & 0b0000_0111)
		imm5 := ((first & 0b0000_0111) << 2) | ((last & 0b1100_0000) >> 6)
		imm32 := uint32(imm5) << 2
		out.text = fmt.Sprintf("str %s, [%s, #%v]", reg(rt), reg(rn), imm32)
	}

	if first&0b1111_1000 == 0 { // lsl rd, rm, #imm5
		imm5 := ((first & 0b0000_0111) << 2) | ((last & 0b1100_0000) >> 6)
		rm := (last & 0b0011_1000) >> 3
		rd := (last & 0b0000_0111)
		out.text = fmt.Sprintf("lsl %s, %s, #%v", reg(rd), reg(rm), imm5)
	}

	if first&0b1111_1110 == 0b0001_1110 { // sub rd, rn, #imm3
		imm3 := ((first & 0b0000_0001) << 2) | ((last & 0b1100_0000) >> 6)
		rn := (last & 0b0011_1000) >> 3
		rd := (last & 0b0000_0111)

		out.text = fmt.Sprintf("sub %s, %s, #%v", reg(rd), reg(rn), imm3)
	}

	if first&0b1111_1000 == 0b0011_1000 { // sub rdn, #imm8
		rdn := first & 0b0000_0111
		out.text = fmt.Sprintf("sub %s, #%v", reg(rdn), last)
	}

	if first&0b1111_1000 == 0b0010_1000 { // cmp rn, #imm8
		rn := first & 0b0000_0111
		out.text = fmt.Sprintf("cmp %s, #%v", reg(rn), last)
	}

	if first == 0b0100_0111 && last&0b1000_0000 == 0 { // bx rm
		rm := last & 0b0111_1000 >> 3
		out.text = fmt.Sprintf("bx %s", reg(rm))
	}

	return true
}

func reg(r uint8) string {
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

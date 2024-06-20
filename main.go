package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
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
	blocks := []*uf2block{}
	rb := newReadBuffer(bytes)

	out := readChunk(rb)
	for out != nil {
		blocks = append(blocks, out)

		fmt.Print(out.Header())
		fmt.Println()
		fmt.Print(out.HexPayload())
		fmt.Println()

		out = readChunk(rb)
	}

	maps := joinBlocks(blocks)
	for _, m := range maps {
		fmt.Printf("\n----------- REGION 0x%04X  %v bytes-----------\n", m.addr, len(m.contents))
		fmt.Print(Disassemble(m))
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

type memoryMap struct {
	addr     uint32
	contents []byte
}

func newMMap(block *uf2block) *memoryMap {
	return &memoryMap{
		addr:     block.addr,
		contents: block.payload,
	}
}

func (this *memoryMap) append(block *uf2block) {
	this.contents = append(this.contents, block.payload...)
}

func (this *memoryMap) precedes(block *uf2block) bool {
	return this.addr+uint32(len(this.contents)) == block.addr
}

// not all blocks are adjacent...
func joinBlocks(blocks []*uf2block) []*memoryMap {
	if len(blocks) == 0 {
		return []*memoryMap{}
	}
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].addr < blocks[j].addr
	})

	out := []*memoryMap{}
	currMap := newMMap(blocks[0])
	for i := 1; i < len(blocks); i++ {
		if currMap.precedes(blocks[i]) {
			currMap.append(blocks[i])
		} else {
			out = append(out, currMap)
			currMap = newMMap(blocks[i])
		}
	}
	out = append(out, currMap)
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

func (this *uf2block) Header() string {
	out := ""
	out += fmt.Sprintf("\t%v", this.flags)
	if this.flags.FamilyIDPresent {
		out += fmt.Sprintf("\t\t%08X\n", this.something)
	} else {
		out += fmt.Sprintf("\t\t%08X\n", this.something)
	}
	out += fmt.Sprintf("\t%v bytes\t%08X\t%v/%v",
		this.payloadSize,
		this.addr,
		this.seqBlockNum+1,
		this.totBlockNum,
	)
	return out
}

func (this *uf2block) HexPayload() string {
	return hexPrint(this.payload)
}
func strchunk(chunk []byte) string {
	if len(chunk) == 2 {
		return fmt.Sprintf("    %02X%02X", chunk[1], chunk[0])
	} else if len(chunk) == 4 {
		return fmt.Sprintf("%02X%02X%02X%02X", chunk[3], chunk[2], chunk[1], chunk[0])
	} else {
		panic(len(chunk))
	}
}

type instr struct {
	text  string
	size  uint32
	chunk []byte
}

const (
	bits15_14 uint16 = 0b1100_0000_0000_0000
	bits15_12 uint16 = 0b1111_0000_0000_0000
	bits15_11 uint16 = 0b1111_1000_0000_0000
	bits15_10 uint16 = 0b1111_1100_0000_0000
	bits15_9  uint16 = 0b1111_1110_0000_0000
	bits15_8  uint16 = 0b1111_1111_0000_0000
	bits15_7  uint16 = 0b1111_1111_1000_0000
	bits15_6  uint16 = 0b1111_1111_1100_0000
	bits15_5  uint16 = 0b1111_1111_1110_0000
	bits15_4  uint16 = 0b1111_1111_1111_0000

	bits10_0 uint16 = 0b0000_0111_1111_1111
	bits9_0  uint16 = 0b0000_0011_1111_1111
	bits7_0  uint16 = 0b0000_0000_1111_1111
	bits6_0  uint16 = 0b0000_0000_0111_1111

	bits11_8 uint16 = 0b0000_1111_0000_0000
	bits10_8 uint16 = 0b0000_0111_0000_0000
	bits10_6 uint16 = 0b0000_0111_1100_0000
	bits8_6  uint16 = 0b0000_0001_1100_0000
	bits6_3  uint16 = 0b0000_0000_0111_1000
	bits5_3  uint16 = 0b0000_0000_0011_1000
	bits3_0  uint16 = 0b0000_0000_0000_1111
	bits2_0  uint16 = 0b0000_0000_0000_0111

	bit13 uint16 = 0b0010_0000_0000_0000
	bit12 uint16 = 0b0001_0000_0000_0000
	bit11 uint16 = 0b0000_1000_0000_0000
	bit10 uint16 = 0b0000_0100_0000_0000
	bit9  uint16 = 0b0000_0010_0000_0000
	bit8  uint16 = 0b0000_0001_0000_0000
	bit7  uint16 = 0b0000_0000_1000_0000
)

func Disassemble(m *memoryMap) string {
	out := ""
	rb := newReadBuffer(m.contents)

	var instrOut instr
	startAddr := m.addr
	for decodeInstr(rb, &instrOut) {
		out += fmt.Sprintf("%08X", startAddr) +
			" " + strchunk(instrOut.chunk) +
			"\t" + instrOut.text + "\n"
		startAddr += instrOut.size
	}

	return out
}

func decodeInstr(rb *ReadBuffer, out *instr) bool {
	hw, ok := rb.getU16()
	if !ok {
		return false
	}
	first := uint8((hw >> 8) & 0xFF)
	last := uint8(hw & 0xFF)

	out.chunk = []byte{first, last} // little endian
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
		out.text = fmt.Sprintf("ADDS %v, %v, #%01X", reg(rd), reg(rn), imm3)
	}

	if hw&bits15_11 == 0b0011_0000_0000_0000 { // ADDS <Rdn>, #<imm8>
		imm8 := hw & bits7_0
		rdn := (hw & bits10_8) >> 8
		out.text = fmt.Sprintf("ADDS %v, #%02X", reg(rdn), imm8)
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
		out.text = fmt.Sprintf("ADD %v, SP, #%02X", reg(rd), imm8)
	}

	if hw&bits15_7 == 0b1011_0000_0000_0000 { // ADD SP, SP, #<imm7>
		imm7 := (hw & bits6_0) << 2
		out.text = fmt.Sprintf("ADD SP, SP, #%02X", imm7)
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
		out.text = fmt.Sprintf("ADR %v, PC, #%02X", reg(rd), imm8)
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
		out.text = fmt.Sprintf("ASRS %v, %v, #%02X", reg(rd), reg(rm), shift_n)
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
			out.text = fmt.Sprintf("UDF #%02X", imm8)
		} else if c == 0b1111 {
			out.text = fmt.Sprintf("SVC #%02X", imm8)
		} else {
			offset := int32(int8(imm8)) << 1
			out.text = fmt.Sprintf("B%v [PC, #%02X]", cond(c), offset)
		}
	}

	if hw&bits15_11 == 0b1110_0000_0000_0000 { // b <label>
		imm11 := hw & bits10_0
		imm32 := int32(int16(imm11<<5) >> 4)
		out.text = fmt.Sprintf("B [PC, #%02X]", imm32)
	}

	if hw&bits15_6 == 0b0100_0011_1000_0000 { // BICS <rdn>, <rm>
		rm := (hw & bits5_3) >> 3
		rdn := hw & bits2_0
		out.text = fmt.Sprintf("BICS %v, %v", reg(rdn), reg(rm))
	}

	if hw&bits15_8 == 0b1011_1110_0000_0000 { // BKPT #<imm8>
		imm8 := hw & bits7_0
		out.text = fmt.Sprintf("BKPT #%02X", imm8)
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
		out.text = fmt.Sprintf("CMP %v, #%02X", reg(rn), imm8)
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

	if hw&bits15_6 == 0b0100_0000_0100_0000 { // EORS <rdn>, <rm>
		rm := (hw & bits5_3) >> 3
		rdn := hw & bits2_0
		out.text = fmt.Sprintf("EORS %v, %v", reg(rdn), reg(rm))
	}
	if hw&bits15_11 == 0b1100_1000_0000_0000 { // LDM <rn>, <registers>
		rn := (hw & bits10_8) >> 8
		list := (hw & bits7_0)
		out.text = fmt.Sprintf("LDM %v, %v", reg(rn), reglist(list))
	}

	if hw&bits15_11 == 0b0110_1000_0000_0000 { // LDR <rt>, [<rn>, #<imm5>]
		rn := (hw & bits5_3) >> 3
		rt := hw & bits2_0
		imm5 := (hw & bits10_6) >> 4
		out.text = fmt.Sprintf("LDR %v, [%v, #%02X]", reg(rt), reg(rn), imm5)
	}

	if hw&bits15_11 == 0b1001_1000_0000_0000 { // LDR <Rt>, [SP, #imm8]
		rt := (hw & bits10_8) >> 8
		imm8 := (hw & bits7_0) << 2
		out.text = fmt.Sprintf("LDR %v, [SP, #%02X]", reg(rt), imm8)
	}

	if hw&bits15_11 == 0b0100_1000_0000_0000 { // LDR <rt>, <label>
		rt := (hw & bits10_8) >> 8
		imm8 := hw & bits7_0 << 2
		out.text = fmt.Sprintf("LDR %v, [PC, #%02X]", reg(rt), imm8)
	}

	if hw&bits15_9 == 0b0101_1000_0000_0000 { // LDR <rt>, [<rn, <rm>]
		rm := (hw & bits8_6) >> 6
		rn := (hw & bits5_3) >> 3
		rt := hw & bits2_0
		out.text = fmt.Sprintf("LDR %v, [%v, %v]", reg(rt), reg(rn), reg(rm))
	}

	if hw&bits15_11 == 0b0111_1000_0000_0000 { // LDRB <rt>, [<rn>, #<imm5>]
		rn := (hw & bits5_3) >> 3
		rt := hw & bits2_0
		imm5 := (hw & bits10_6) >> 6

		out.text = fmt.Sprintf("LDRB %v, [%v, #%02X]", reg(rt), reg(rn), imm5)
	}

	if hw&bits15_9 == 0b0101_1100_0000_0000 { // LDRB <rt>, [<rn>, <rm>]
		rm := (hw & bits8_6) >> 6
		rn := (hw & bits5_3) >> 3
		rt := hw & bits2_0

		out.text = fmt.Sprintf("LDRB %v, [%v, %v]", reg(rt), reg(rn), reg(rm))
	}

	if hw&bits15_11 == 0b1000_1000_0000_0000 { // LDRH <rt>, [<rn>, #<imm5>]
		rn := (hw & bits5_3) >> 3
		rt := hw & bits2_0
		imm5 := (hw & bits10_6) >> 5

		out.text = fmt.Sprintf("LDRH %v, [%v, #%02X]", reg(rt), reg(rn), imm5)
	}

	if hw&bits15_9 == 0b0101_1010_0000_0000 { // LDRH <rt>, [<rn>, <rm>]
		rm := (hw & bits8_6) >> 6
		rn := (hw & bits5_3) >> 3
		rt := hw & bits2_0

		out.text = fmt.Sprintf("LDRH %v, [%v, %v]", reg(rt), reg(rn), reg(rm))
	}

	if hw&bits15_9 == 0b0101_0110_0000_0000 { // LDRSB <rt>, [<rn>, <rm>]
		rm := (hw & bits8_6) >> 6
		rn := (hw & bits5_3) >> 3
		rt := hw & bits2_0

		out.text = fmt.Sprintf("LDRSB %v, [%v, %v]", reg(rt), reg(rn), reg(rm))
	}

	if hw&bits15_9 == 0b0101_1110_0000_0000 { // LDRSH <rt>, [<rn>, <rm>]
		rm := (hw & bits8_6) >> 6
		rn := (hw & bits5_3) >> 3
		rt := hw & bits2_0

		out.text = fmt.Sprintf("LDRSH %v, [%v, %v]", reg(rt), reg(rn), reg(rm))
	}

	if hw&bits15_11 == 0b0000_0000_0000_0000 { // LSLS <rd>, <rm>, #<imm5>
		rm := (hw & bits5_3) >> 3
		rd := hw & bits2_0
		imm5 := (hw & bits10_6) >> 6

		if imm5 == 0b00000 { // MOV
			out.text = fmt.Sprintf("MOV %v, %v", reg(rd), reg(rm))
		} else {
			out.text = fmt.Sprintf("LSLS %v, %v, #%02X", reg(rd), reg(rm), imm5)
		}
	}

	if hw&bits15_6 == 0b0100_0000_1000_0000 { // LSLS <rdn>, <rm>
		rm := (hw & bits5_3) >> 3
		rdn := hw & bits2_0
		out.text = fmt.Sprintf("LSLS %v, %v", reg(rdn), reg(rm))
	}

	if hw&bits15_11 == 0b0000_1000_0000_0000 { // LSRS <rd>, <rm>, #<imm5>
		rm := (hw & bits5_3) >> 3
		rd := hw & bits2_0
		imm5 := (hw & bits10_6) >> 6
		shift_n := imm5
		if imm5 == 0 {
			shift_n = 32
		}
		out.text = fmt.Sprintf("LSRS %v, %v, #%02X", reg(rd), reg(rm), shift_n)
	}

	if hw&bits15_6 == 0b0100_0000_1100_0000 { // LSRS <rdn>, <rm>
		rm := (hw & bits5_3) >> 3
		rdn := hw & bits2_0
		out.text = fmt.Sprintf("LSRS %v, %v", reg(rdn), reg(rm))
	}

	if hw&bits15_11 == 0b0010_0000_0000_0000 { // MOVS <rd>, #<imm8>
		rd := (hw & bits10_8) >> 8
		imm8 := hw & bits7_0
		out.text = fmt.Sprintf("MOV %v, #%02X", reg(rd), imm8)
	}

	if hw&bits15_8 == 0b0100_0110_0000_0000 { // MOV <rd>, <rm>
		D := (hw & bit7) >> 4
		rd := (hw & bits2_0)
		d := D | rd

		rm := (hw & bits6_3) >> 3
		out.text = fmt.Sprintf("MOV %v, %v", reg(d), reg(rm))
	}

	if hw&bits15_6 == 0b0100_0011_0100_0000 { // MULS <rdm>, <rn>, <rdm>
		rn := (hw & bits5_3) >> 3
		rdm := hw & bits2_0
		out.text = fmt.Sprintf("MULS %v, %v, %v", reg(rdm), reg(rn), reg(rdm))
	}

	if hw&bits15_6 == 0b0100_0011_1100_0000 { // MVN <rd>, <rm>
		rm := (hw & bits5_3) >> 3
		rd := hw & bits2_0
		out.text = fmt.Sprintf("MVNS %v, %v", reg(rd), reg(rm))
	}

	if hw&bits15_6 == 0b0100_0010_0100_0000 { // NEGS <rd>, <rm>
		rn := (hw & bits5_3) >> 3
		rd := hw & bits2_0
		out.text = fmt.Sprintf("NEGS %v, %v", reg(rd), reg(rn))
	}

	if hw&bits15_6 == 0b0100_0011_0000_0000 { // ORRS <rdn>, <rm>
		rdn := hw & bits2_0
		rm := (hw & bits5_3) >> 3
		out.text = fmt.Sprintf("ORRS %v, %v", reg(rdn), reg(rm))
	}

	if hw&bits15_9 == 0b1011_1100_0000_0000 { // POP <registers>
		P := (hw & bit8) << 7
		list := P | (hw & bits7_0)
		out.text = fmt.Sprintf("POP %v", reglist(list))
	}

	if hw&bits15_9 == 0b1011_0100_0000_0000 { // PUSH <registers>
		M := (hw & bit8) << 6
		list := M | (hw & bits7_0)
		out.text = fmt.Sprintf("PUSH %v", reglist(list))
	}

	if hw&bits15_6 == 0b1011_1010_0000_0000 { // REV <rd>, <rm>
		rm := (hw & bits5_3) >> 3
		rd := (hw & bits2_0)
		out.text = fmt.Sprintf("REV %v, %v", reg(rd), reg(rm))
	}

	if hw&bits15_6 == 0b1011_1010_0100_0000 { // REV16 <rd>, <rm>
		rm := (hw & bits5_3) >> 3
		rd := (hw & bits2_0)
		out.text = fmt.Sprintf("REV16 %v, %v", reg(rd), reg(rm))
	}

	if hw&bits15_6 == 0b1011_1010_1100_0000 { // REVSH <rd>, <rm>
		rm := (hw & bits5_3) >> 3
		rd := (hw & bits2_0)
		out.text = fmt.Sprintf("REVSH %v, %v", reg(rd), reg(rm))
	}

	if hw&bits15_6 == 0b0100_0001_1100_0000 { // RORS <rdn>, <rm>
		rm := (hw & bits5_3) >> 3
		rdn := (hw & bits2_0)
		out.text = fmt.Sprintf("RORS %v, %v", reg(rdn), reg(rm))
	}

	if hw&bits15_6 == 0b0100_0001_1000_0000 { // SBCS <rdn>, <rm>
		rm := (hw & bits5_3) >> 3
		rdn := (hw & bits2_0)
		out.text = fmt.Sprintf("SBCS %v, %v", reg(rdn), reg(rm))
	}

	if hw == 0b1011_1111_0100_0000 { // SEV
		out.text = "SEV"
	}

	if hw&bits15_11 == 0b1100_0000_0000_0000 { // STM <rn>!, <registers>
		rn := (hw & bits10_8) >> 8
		list := hw & bits7_0
		out.text = fmt.Sprintf("STM %v, %v", reg(rn), reglist(list))
	}

	if hw&bits15_11 == 0b0110_0000_0000_0000 { // STR <rt>, [<rn>, #<imm5>]
		imm5 := (hw & bits10_6) >> 4
		rn := (hw & bits5_3) >> 3
		rt := hw & bits2_0
		out.text = fmt.Sprintf("STR %v, [%v, #%02X]", reg(rt), reg(rn), imm5)
	}

	if hw&bits15_11 == 0b1001_0000_0000_0000 { // STR <rt>, [SP, #<imm8>]
		rt := (hw & bits10_8) >> 8
		imm8 := (hw & bits7_0) << 2
		out.text = fmt.Sprintf("STR %v, [SP, #%02X]", reg(rt), imm8)
	}

	if hw&bits15_9 == 0b0101_0000_0000_0000 { // STR <rt>, <rn>, <rm>
		rm := (hw & bits8_6) >> 6
		rn := (hw & bits5_3) >> 3
		rt := hw & bits2_0

		out.text = fmt.Sprintf("STR %v, %v, %v", reg(rt), reg(rn), reg(rm))
	}

	if hw&bits15_11 == 0b0111_0000_0000_0000 { // STRB <rt>, [<rn>, #<imm5>]
		imm5 := (hw & bits10_6) >> 6
		rn := (hw & bits5_3) >> 3
		rt := hw & bits2_0

		out.text = fmt.Sprintf("STRB %v, [%v, #%02X]", reg(rt), reg(rn), imm5)
	}

	if hw&bits15_9 == 0b0101_0100_0000_0000 { // STRB <rt>, [<rn>, <rm>]
		rm := (hw & bits8_6) >> 6
		rn := (hw & bits5_3) >> 3
		rt := hw & bits2_0

		out.text = fmt.Sprintf("STRB %v, [%v, %v]", reg(rt), reg(rn), reg(rm))
	}

	if hw&bits15_11 == 0b1000_0000_0000_0000 { // STRH <rt>, [<rn>, #<imm5>]
		imm5 := (hw & bits10_6) >> 5
		rn := (hw & bits5_3) >> 3
		rt := hw & bits2_0

		out.text = fmt.Sprintf("STRH %v, [%v, #%02X]", reg(rt), reg(rn), imm5)
	}

	if hw&bits15_9 == 0b0101_0010_0000_0000 { // STRH <rt>, [<rn>, <rm>]
		rm := (hw & bits8_6) >> 6
		rn := (hw & bits5_3) >> 3
		rt := hw & bits2_0

		out.text = fmt.Sprintf("STRH %v, [%v, %v]", reg(rt), reg(rn), reg(rm))
	}

	if hw&bits15_9 == 0b0001_1110_0000_0000 { // SUBS <rd>, <rn>, #<imm3>
		imm3 := (hw & bits8_6) >> 6
		rn := (hw & bits5_3) >> 3
		rd := hw & bits2_0

		out.text = fmt.Sprintf("SUBS %v, %v, #%02X", reg(rd), reg(rn), imm3)
	}

	if hw&bits15_11 == 0b0011_1000_0000_0000 { // SUBS <rdn>, #<imm8>
		imm8 := hw & bits7_0
		rdn := (hw & bits10_8) >> 8

		out.text = fmt.Sprintf("SUBS %v, #%02X", reg(rdn), imm8)
	}

	if hw&bits15_9 == 0b0001_1010_0000_0000 { // SUBS <rd>, <rn>, <rm>
		rm := (hw & bits8_6) >> 6
		rn := (hw & bits5_3) >> 3
		rd := hw & bits2_0
		out.text = fmt.Sprintf("SUBS %v, %v, %v", reg(rd), reg(rn), reg(rm))
	}

	if hw&bits15_7 == 0b1011_0000_1000_0000 { // SUB SP, SP, #<imm7>
		imm7 := hw & bits6_0
		out.text = fmt.Sprintf("SUB SP, SP, #%02X", imm7)
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

	if hw&bits15_6 == 0b0100_0010_0000_0000 { // TST <rn>, <rm>
		rm := (hw & bits5_3) >> 3
		rn := hw & bits2_0
		out.text = fmt.Sprintf("TST %v, %v", reg(rn), reg(rm))
	}

	if hw&bits15_6 == 0b1011_0010_1100_0000 { // UXTB <rd> , <rm>
		rm := (hw & bits5_3) >> 3
		rd := hw & bits2_0
		out.text = fmt.Sprintf("UTXB %v, %v", reg(rd), reg(rm))
	}

	if hw&bits15_6 == 0b1011_0010_1000_0000 { // UXTH <rd> , <rm>
		rm := (hw & bits5_3) >> 3
		rd := hw & bits2_0
		out.text = fmt.Sprintf("UTXH %v, %v", reg(rd), reg(rm))
	}

	if hw == 0b1011_1111_0010_0000 { // WFE
		out.text = "WFE"
	}

	if hw == 0b1011_1111_0011_0000 { // WFI
		out.text = "WFI"
	}

	if hw == 0b1011_1111_0001_0000 { // YIELD
		out.text = "YIELD"
	}

	if hw&bits15_11 == 0b1111_0000_0000_0000 { // 32 bit instruction
		hw2, ok := rb.getU16()
		if !ok {
			out.text = fmt.Sprintf("32 bit instruction")
			return false
		}

		first := uint8((hw2 >> 8) & 0xFF)
		last := uint8(hw2 & 0xFF)

		out.chunk = append([]byte{first, last}, out.chunk...)
		out.size += 2
		out.text = fmt.Sprintf("32 bit instruction")
		if hw2&(bits15_14|bit12) == 0b1101_0000_0000_0000 { // BL <label>
			imm10 := uint32(hw & bits9_0)
			S := (hw & bit10) >> 10

			imm11 := uint32(hw2 & bits10_0)
			J1 := (hw2 & bit13) >> 13
			J2 := (hw2 & bit11) >> 11

			I1 := uint32(^(J1 ^ S)) // why??
			I2 := uint32(^(J2 ^ S))

			u24 := (uint32(S) << 23) | (I1 << 22) | (I2 << 21) | (imm10 << 11) | imm11
			i32 := int32(u24<<8) >> 8

			imm32 := i32 << 1

			out.text = fmt.Sprintf("BL [PC, #%04X]", imm32)
		}

		if hw == 0b1111_0011_1011_1111 { // DMB / DSB
			if hw2&bits15_4 == 0b1000_1111_0101_0000 {
				out.text = "DMB"
			} else if hw2&bits15_4 == 0b1000_1111_0100_0000 {
				out.text = "DSB"
			} else if hw2&bits15_4 == 0b1000_1111_0110_0000 {
				out.text = "ISB"
			} else {
				out.text = "!!!"
			}
		}

		if hw == 0b1111_0011_1110_1111 { // MRS <rd>, <spec_reg>
			if hw2&bits15_12 == 0b1000_0000_0000_0000 {
				rd := (hw2 & bits15_12) >> 12
				SYSm := uint8(hw2 & bits7_0)
				out.text = fmt.Sprintf("MRS %v, <%08b>", reg(rd), SYSm)
			} else {
				out.text = "!!!"
			}
		}

		if hw&bits15_4 == 0b1111_0011_1000_0000 { // MSR <spec_reg>, <rn>
			if hw2&bits15_8 == 0b1000_1000_0000_0000 {
				rn := hw & bits3_0
				SYSm := uint8(hw & bits7_0)
				out.text = fmt.Sprintf("MSR <%08b>, %v", SYSm, reg(rn))
			} else {
				out.text = "!!!"
			}
		}
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

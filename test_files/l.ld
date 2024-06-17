MEMORY {
    SRAM(rx) : ORIGIN = 0x20040000, LENGTH = 512
}

SECTIONS {
    . = ORIGIN(SRAM);
    .text : {
        *(.vectors)
        *(.entry)
        *(.text)
    } >SRAM
}

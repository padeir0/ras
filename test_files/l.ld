MEMORY {
    SRAM(rx) : ORIGIN = 0x20040000, LENGTH = 2048
}

SECTIONS {
    . = ORIGIN(SRAM);
    .text : {
        *(.vectors)
        *(.entry)
        *(.text)
    } >SRAM
}

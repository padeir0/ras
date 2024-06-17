	.cpu cortex-m0plus
	.thumb
	.syntax unified

/* vector table */
	.section .vectors, "ax"
	.align 2
	.global _vectors
_vectors:
	.word 0x20001000
	.word _reset

/* reset handler */
	.thumb_func
	.global _reset
_reset:
	ldr r0, =0x20001000  ;@ Stack Pointer
	mov sp, r0

	mov r8, r8
	mov r8, r8

	adcs r0, r1
	adcs r1, r2
	adcs r2, r3
	adcs r3, r4

	adds r0, #16
	adds r1, #32
	adds r2, #64
	adds r3, #128

	adds r0, r1, #2
	adds r1, r2, #3
	adds r2, r3, #4
	adds r3, r4, #5

	adds r0, r1, r2
	adds r2, r3, r4
	adds r4, r5, r6
	adds r7, r0, r1

	add r0, r1
	add r1, r2
	add r2, r3
	add sp, r4

	add r0, sp, #32
	add r1, sp, #64
	add r2, sp, #128
	add r3, sp, #256

	add sp, sp, #16
	add sp, sp, #32
	add sp, sp, #64
	add sp, sp, #128

	add r0, sp, r0
	add r1, sp, r1
	add r2, sp, r2
	add r3, sp, r3

	add sp, r0
	add sp, r1
	add sp, r2
	add sp, r3
_l0:
	add r0, pc, #32
	add r1, pc, #64
	add r2, pc, #128
	add r3, pc, #256
_l1:
	ands r0, r1
	ands r1, r2
	ands r2, r3
	ands r3, r4
_l2:
	asrs r0, r1, #4
	asrs r1, r2, #8
	asrs r2, r3, #16
	asrs r3, r4, #32

_l3:
	asrs r0, r1
	asrs r1, r2
	asrs r2, r3
	asrs r3, r4

	beq _l0
	bne _l1
	bcs _l2
	bcc _l3
	bmi _l0
	bpl _l1
	bvs _l2
	bvc _l3
	bhi _l0
	bls _l1
	bge _l2
	blt _l3
	bgt _l0
	ble _l1

	b _l0
	b _l1
	b _l2
	b _l3

	bics r0, r1
	bics r1, r2
	bics r2, r3
	bics r3, r4

	bkpt #32
	bkpt #64
	bkpt #128
	bkpt #255

	bl _l0
	bl _l1
	bl _l2
	bl _l3

	blx r0
	blx r1
	blx r2
	blx r3

	bx r0
	bx r1
	bx r2
	bx r3

	cmn r0, r1
	cmn r1, r2
	cmn r2, r3
	cmn r3, r4

	cmp r0, #32
	cmp r1, #64
	cmp r2, #128
	cmp r3, #255

	cmp r0, r8
	cmp r1, r9
	cmp r2, r10
	cmp r3, r11

	DMB
	DMB #15

	DSB
	DSB #15

	/* REMAINING FROM */
	/* A6.7.23 EOR (register) */
.align 4

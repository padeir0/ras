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

	adcs r0, r1 @ done
	adcs r1, r2
	adcs r2, r3
	adcs r3, r4

	adds r0, #16 @ done
	adds r1, #32
	adds r2, #64
	adds r3, #128

	adds r0, r1, #2 @ done
	adds r1, r2, #3
	adds r2, r3, #4
	adds r3, r4, #5

	adds r0, r1, r2 @ done
	adds r2, r3, r4
	adds r4, r5, r6
	adds r7, r0, r1

	add r0, r1 @ done
	add r1, r2
	add r2, r3
	add sp, r4

	add r0, sp, #32 @ done
	add r1, sp, #64
	add r2, sp, #128
	add r3, sp, #256

	add sp, sp, #16 @ done
	add sp, sp, #32
	add sp, sp, #64
	add sp, sp, #128

	add r0, sp, r0 @ done
	add r1, sp, r1
	add r2, sp, r2
	add r3, sp, r3

	add sp, r0 @ done
	add sp, r1
	add sp, r2
	add sp, r3
_l0:
	add r0, pc, #32 @ done
	add r1, pc, #64
	add r2, pc, #128
	add r3, pc, #256
_l1:
	ands r0, r1 @ done
	ands r1, r2
	ands r2, r3
	ands r3, r4
_l2:
	asrs r0, r3, #0 @ done
	asrs r1, r2, #4
	asrs r2, r3, #16
	asrs r3, r4, #31
	asrs r1, r2, #32
_l3:
	asrs r0, r1 @ done
	asrs r1, r2
	asrs r2, r3
	asrs r3, r4

	beq _l0 @done
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
	DSB

	eors r1, r2
	eors r2, r3
	eors r3, r4
	eors r4, r5

	ISB

	ldm r0!, {r1, r2, r3, r4}
	ldm r0!, {r4, r5, r6, r7}
	ldm r1!, {r5, r7}
	ldm r1!, {r2, r3, r4, r5, r6, r7}

	ldr r3, [r1, #4]
	ldr r4, [r2, #8]
	ldr r5, [r3, #16]
	ldr r6, [r4, #32]

	ldr r3, [sp, #16]
	ldr r4, [sp, #32]
	ldr r5, [sp, #64]
	ldr r6, [sp, #128]

	ldr r0, =0xdeadbeef 
	ldr r1, =0xcafebabe
	ldr r2, =0xdeadbabe
	ldr r3, =0xbabebeef

	ldr r0, [r1, r2]
	ldr r2, [r3, r4]
	ldr r4, [r5, r6]
	ldr r5, [r6, r7]

	ldrb r0, [r1, #4]
	ldrb r1, [r2, #8]
	ldrb r2, [r3, #16]
	ldrb r3, [r4, #31]

	ldrb r0, [r1, r2]
	ldrb r2, [r3, r4]
	ldrb r4, [r5, r6]
	ldrb r5, [r6, r7]

	ldrh r0, [r1, #4]
	ldrh r1, [r2, #8]
	ldrh r2, [r3, #16]
	ldrh r3, [r4, #30]

	ldrh r0, [r1, r2]
	ldrh r2, [r3, r4]
	ldrh r4, [r5, r6]
	ldrh r5, [r6, r7]

	ldrsb r0, [r1, r2]
	ldrsb r2, [r3, r4]
	ldrsb r4, [r5, r6]
	ldrsb r5, [r6, r7]

	ldrsh r0, [r1, r2]
	ldrsh r2, [r3, r4]
	ldrsh r4, [r5, r6]
	ldrsh r5, [r6, r7]

	lsls r0, r1, #4 
	lsls r1, r2, #8
	lsls r2, r3, #16
	lsls r3, r4, #31

	lsls r0, r1
	lsls r1, r2
	lsls r2, r3
	lsls r3, r4

	lsrs r0, r1, #4
	lsrs r1, r2, #8
	lsrs r2, r3, #16
	lsrs r3, r4, #31

	lsrs r0, r1
	lsrs r1, r2
	lsrs r2, r3
	lsrs r3, r4

	movs r0, #32 
	movs r1, #64
	movs r2, #128
	movs r3, #255

	mov r0, r7 
	mov r8, r1
	mov r2, r9
	mov r10, r3

	movs r0, r1 
	movs r1, r2
	movs r2, r3
	movs r3, r4

	mrs r0, apsr
	mrs r1, iapsr
	mrs r2, eapsr
	mrs r3, xpsr
	mrs r4, ipsr
	mrs r5, epsr
	mrs r6, iepsr
	mrs r7, msp
	mrs r0, psp
	mrs r1, primask
	mrs r2, control

	msr apsr_nzcvq, r0
	msr iapsr_nzcvq, r1
	msr eapsr_nzcvq, r2
	msr xpsr_nzcvq, r3
	msr ipsr, r4
	msr epsr, r5
	msr iepsr, r6
	msr msp, r7
	msr psp, r0
	msr primask, r1
	msr control, r2

	muls r0, r1, r0
	muls r2, r3, r2
	muls r3, r4, r3
	muls r4, r5, r4

	mvns r0, r1
	mvns r2, r1
	mvns r3, r2
	mvns r4, r3

	nop 

	orrs r0, r1 
	orrs r1, r2
	orrs r2, r3
	orrs r3, r4

	pop {r0, r1, r2, r3, r4, r5}
	pop {r3, r4, r5}
	pop {r0, r1, pc}
	pop {pc}

	push {r0, r1, r2, r3, r4, r5}
	push {r3, r4, r5}
	push {r0, r1, lr}
	push {lr}

	rev r0, r1
	rev r1, r2
	rev r2, r3
	rev r3, r4

	rev16 r0, r1
	rev16 r1, r2
	rev16 r2, r3
	rev16 r3, r4

	revsh r0, r1
	revsh r1, r2
	revsh r2, r3
	revsh r3, r4

	rors r0, r1
	rors r1, r2
	rors r2, r3
	rors r3, r4

	rsbs r0, r1, #0
	rsbs r1, r2, #0
	rsbs r2, r3, #0
	rsbs r3, r4, #0

	sbcs r0, r1
	sbcs r1, r2
	sbcs r2, r3
	sbcs r3, r4

	sev

	stm r0!, {r1, r2, r3, r4}
	stm r0!, {r4, r5, r6, r7}
	stm r1!, {r5, r7}
	stm r1!, {r2, r3, r4, r5, r6, r7}

	str r0, [r1, #4] 
	str r5, [r2, #8]
	str r6, [r3, #16]
	str r7, [r4, #32]

	str r0, [sp, #32]
	str r5, [sp, #64]
	str r6, [sp, #128]
	str r7, [sp, #256]

	str r0, [r1, r0]
	str r2, [r3, r2]
	str r3, [r4, r3]
	str r4, [r5, r4]

	strb r0, [r1, #4]
	strb r5, [r2, #8]
	strb r6, [r3, #16]
	strb r7, [r4, #31]

	strb r0, [r1, r0]
	strb r2, [r3, r2]
	strb r3, [r4, r3]
	strb r4, [r5, r4]

	strh r0, [r1, #4]
	strh r5, [r2, #8]
	strh r6, [r3, #16]
	strh r7, [r4, #32]

	strh r0, [r1, r0]
	strh r2, [r3, r2]
	strh r3, [r4, r3]
	strh r4, [r5, r4]

	subs r0, r1, #0 
	subs r1, r2, #1
	subs r3, r4, #2
	subs r4, r5, #3

	subs r0, #32 
	subs r1, #64
	subs r3, #128
	subs r4, #255

	subs r0, r1, r0
	subs r2, r3, r2
	subs r3, r4, r3
	subs r4, r5, r4

	sub sp, sp, #16
	sub sp, sp, #32
	sub sp, sp, #64
	sub sp, sp, #128

	svc #32
	svc #64
	svc #128
	svc #255

	sxtb r0, r1
	sxtb r2, r3
	sxtb r3, r4
	sxtb r4, r5

	sxth r0, r1
	sxth r2, r3
	sxth r3, r4
	sxth r4, r5

	tst r0, r1
	tst r2, r3
	tst r3, r4
	tst r4, r5

	udf #32
	udf #64
	udf #128
	udf #255

	uxtb r0, r1
	uxtb r2, r3
	uxtb r3, r4
	uxtb r4, r5

	uxth r0, r1
	uxth r2, r3
	uxth r3, r4
	uxth r4, r5

	wfe
	wfi
	yield
.align 4

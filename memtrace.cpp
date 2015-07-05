#include <stdio.h>
#include <fcntl.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>

#include <asm/unistd.h>         // __NR_futex
#include <linux/futex.h>        // FUTEX_WAIT

#include "pin.H"

struct record 
{
        ADDRINT pc;
        ADDRINT ea;
};

TLS_KEY cbufKey;
BUFFER_ID buf;
int logFD;
PIN_LOCK logLock;

// char mybuf[1024];
// int mybufPos;

// VOID recordWrite(VOID *ip, VOID *addr)
// {
//         fprintf(trace,"%p: W %p\n", ip, addr);
//         const char x[] = "x";
//         if (write(logFD, x, 1) <= 0) {
//                 fprintf(stderr, "write failed\n");
//         }

//         mybuf[mybufPos] = 'x';
//         mybufPos = (mybufPos + 1) % 1024;
// }

VOID
insInstruction(INS ins, VOID *v)
{
        UINT32 memOperands = INS_MemoryOperandCount(ins);

        for (UINT32 i = 0; i < memOperands; i++) {
                if (!INS_MemoryOperandIsWritten(ins, i))
                        continue;

                INS_InsertFillBufferPredicated(
                        ins, IPOINT_BEFORE, buf,
                        IARG_INST_PTR, offsetof(struct record, pc),
                        IARG_MEMORYOP_EA, i, offsetof(struct record, ea),
                        IARG_END);                

                // INS_InsertPredicatedCall(
                //         ins, IPOINT_BEFORE, (AFUNPTR)recordWrite,
                //         IARG_INST_PTR,
                //         IARG_MEMORYOP_EA, i,
                //         IARG_END);                
        }
}

char *
putVarint(char *buf, int64_t n)
{
        uint64_t x = (n << 1) ^ (n >> 63);
        for (; x >= 0x80; x >>= 7) {
                *(buf++) = (x & 0x7F) | 0x80;
        }
        *(buf++) = x;
        return buf;
}

char *
putLEUint64(char *buf, uint64_t n)
{
        for (int i = 0; i < 8; i++, n >>= 8, buf++)
                *buf = (unsigned char)n;
        return buf;
}

size_t
compressRecords(struct record *recs, int n, char *out)
{
        char *outPos = out + 2 * 8;
        ADDRINT prevPC = 0, prevEA = 0;
        for (int i = 0; i < n; i++, recs++) {
                int64_t deltaPC = recs->pc - prevPC;
                int64_t deltaEA = recs->ea - prevEA;
                prevPC = recs->pc;
                prevEA = recs->ea;
                outPos = putVarint(outPos, deltaPC);
                outPos = putVarint(outPos, deltaEA);
        }
        putLEUint64(out, outPos - out);
        putLEUint64(out + 8, n);
        return outPos - out;
}

void
xwrite(int fd, const void *buf, size_t count)
{
        while (count > 0) {
                ssize_t n = write(fd, buf, count);
                if (n <= 0) {
                        fprintf(stderr, "log write failed: %s\n",
                                strerror(errno));
                        PIN_ExitProcess(1);
                }
                count -= n;
                buf = (char*)buf + n;
        }
}

VOID *
flushBuf(BUFFER_ID id, THREADID tid, const CONTEXT *ctxt, VOID *buf,
         UINT64 numElements, VOID *v)
{
        char *cbuf = (char*)PIN_GetThreadData(cbufKey, tid);
        if (cbuf == NULL) {
                cbuf = new char[2 * 8 + numElements * 2 * 10];
                PIN_SetThreadData(cbufKey, cbuf, tid);
        }
        size_t bytes = compressRecords((struct record*)buf, numElements, cbuf);
        PIN_GetLock(&logLock, tid);
        xwrite(logFD, cbuf, bytes);
        PIN_ReleaseLock(&logLock);
        return buf;
}

void
freeCbuf(void *cbuf)
{
        delete[] (char*)cbuf;
}

VOID Fini(INT32 code, VOID *v) 
{
        printf("FINI\n");
}

struct timespec futexTimeout = {
        .tv_sec = 1000,
        .tv_nsec = 0,
};

VOID
insSyscall(THREADID threadIndex, CONTEXT *ctxt, SYSCALL_STANDARD std, VOID *v) 
{
        // Workaround: When a thread calls exit_group, PIN tries to
        // kick all threads out of system calls and exit them, but it
        // can't kick threads out of an untimed futex wait. Most
        // likely, this is because Go runs with syscall restarting
        // enabled, which means a signal won't kick an untimed futex
        // out of the syscall. Since futex wait is always allowed to
        // return spuriously, we can transform untimed waits into
        // timed waits. It doesn't matter what this timeout is; it
        // just needs to have one (probably because that makes it
        // interruptible with a signal).
        if (PIN_GetSyscallNumber(ctxt, std) == __NR_futex &&
            PIN_GetSyscallArgument(ctxt, std, 1) == FUTEX_WAIT &&
            PIN_GetSyscallArgument(ctxt, std, 3) == 0) {
                //fprintf(stderr, "overriding futex timeout\n");
                PIN_SetSyscallArgument(ctxt, std, 3, (ADDRINT)&futexTimeout);
        }
}

int
main(int argc, char **argv)
{
        enum { NUM_BUF_PAGES = 1024 };

        if (PIN_Init(argc, argv))
                return -1;

        cbufKey = PIN_CreateThreadDataKey(freeCbuf);
        if (cbufKey == -1) {
                fprintf(stderr, "failed to create TLS key for cbuf");
                return 1;
        }

        buf = PIN_DefineTraceBuffer(
                sizeof(struct record), NUM_BUF_PAGES, flushBuf, 0);
        if (buf == BUFFER_ID_INVALID) {
                fprintf(stderr, "could not allocate buffer\n");
                return 1;
        }

        logFD = creat("memtrace.log", 0666);
        if (logFD < 0) {
                fprintf(stderr, "failed to open memtrace.log\n");
                return 1;
        }

        PIN_InitLock(&logLock);

        INS_AddInstrumentFunction(insInstruction, 0);
        // PIN_AddFiniFunction(Fini, 0);

        PIN_AddSyscallEntryFunction(insSyscall, 0);

        PIN_StartProgram();
        return 0;
}

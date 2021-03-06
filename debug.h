#ifndef DEBUG_GUARD
#define DEBUG_GUARD

# include <stdint.h>
# include <stdlib.h>

# ifdef DEBUG
#  define BFMT "%c%c%c%c%c%c%c%c"
#  define B(byte)        \
  (byte & 0x80 ? '1' : '0'), \
  (byte & 0x40 ? '1' : '0'), \
  (byte & 0x20 ? '1' : '0'), \
  (byte & 0x10 ? '1' : '0'), \
  (byte & 0x08 ? '1' : '0'), \
  (byte & 0x04 ? '1' : '0'), \
  (byte & 0x02 ? '1' : '0'), \
  (byte & 0x01 ? '1' : '0')
#  define DEBUGLN(f, ...)                                       \
  do { fprintf (stdout, f "\n", ##__VA_ARGS__); }               \
  while (0)

#  define DEBUG_INSTRUCTION_LOAD() do {                                 \
  char buffer[INSTRUCTION_SIZE+1];                                      \
  buffer[INSTRUCTION_SIZE] = '\0';                                      \
  debug_byte (instr, buffer, INSTRUCTION_SIZE);                         \
  const char *opname = OP_NAME (tmp->rator);                            \
  DEBUGLN ("     INSTR: %s, RATOR: " BFMT                               \
           " (%17s), RAND: " BFMT " (%d)",                              \
           buffer,                                                      \
           B (tmp->rator),                                              \
           opname ? opname : "HALT",                                    \
           B (tmp->u32),                                                \
           UOPERAND0 (tmp));                                            \
  } while (0)

#  define DEBUG_INSTRUCTION_NEXT() do {                                 \
    const char *opname = OP_NAME (pc->rator);                           \
    DEBUGLN ("     RATOR: " BFMT " (%17s), RAND: " BFMT " (%d)",        \
             B (pc->rator), opname ? opname : "HALT",                   \
             B (pc->u32),  UOPERAND0 (pc));                             \
  } while (0)

#  define DEBUG_FAILSTATE() do {                                \
    DEBUGLN ("       FAIL[%s]", i);                             \
    DEBUGLN ("         NEXT: %s", OP_NAME ((*(pc)).rator));     \
  } while (0)

#  define DEBUG_FAILSTATE2() do {                               \
    printf ("       FAIL["); valPrint (l); printf ("]");        \
    DEBUGLN ("         NEXT: %s", OP_NAME ((*(pc)).rator));     \
  } while (0)

#  define DEBUGL(m) do {                        \
    printf ("         %s[", m);                 \
    valPrint(l);                                \
    printf ("]\n");                             \
  } while (0)

#  define DEBUG_STACK() do {                                            \
    DEBUGLN ("         STACK: %p %p", (void *) sp, (void *) m->stack);  \
    for (mStackFrame *_tmp_bt = sp; _tmp_bt > m->stack; _tmp_bt--) {    \
      DEBUGLN ("           [I]: %p %p `%s'",                            \
               (void *) _tmp_bt,                                        \
               (void *) (_tmp_bt - 1)->pc,                              \
               (_tmp_bt - 1)->i);                                       \
    }                                                                   \
  } while (0)

# else  /* TEST */
#  define DEBUGLN(f, ...)
#  define DEBUG_INSTRUCTION_NEXT()
#  define DEBUG_INSTRUCTION_LOAD()
#  define DEBUG_FAILSTATE()
#  define DEBUG_FAILSTATE2()
#  define DEBUGL(m)
#  define DEBUG_STACK()
# endif  /* TEST */

static inline char *debug_byte (uint32_t a, char *buffer, int size) {
  buffer += size - 1;
  for (int i = 31; i >= 0; i--, a >>=1)
    *buffer-- = (a & 1) + '0';
  return buffer;
}

#endif  /* DEBUG_GUARD */

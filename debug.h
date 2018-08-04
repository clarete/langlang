#ifndef DEBUG_H
# include <stdint.h>
# include <stdlib.h>
# ifdef TEST
#  define BY2BIPT "%c%c%c%c%c%c%c%c"
#  define BY2BI(byte)        \
  (byte & 0x80 ? '1' : '0'), \
  (byte & 0x40 ? '1' : '0'), \
  (byte & 0x20 ? '1' : '0'), \
  (byte & 0x10 ? '1' : '0'), \
  (byte & 0x08 ? '1' : '0'), \
  (byte & 0x04 ? '1' : '0'), \
  (byte & 0x02 ? '1' : '0'), \
  (byte & 0x01 ? '1' : '0')
#  define DEBUG(f, ...)                                         \
  do { fprintf (stderr, f "\n", ##__VA_ARGS__); }               \
  while (0)
  
#  define DEBUG_INSTRUCTION() do {                                      \
    uint8_t *instr = (uint8_t *) &instruction;                          \
    int16_t rand16 = operand;                                           \
    const char *opname = OP_NAME (opcode);                              \
    DEBUG ("INSTR: " BY2BIPT "" BY2BIPT                                 \
           ", RATOR: " BY2BIPT " (%13s), RAND: " BY2BIPT " (%d)",       \
           BY2BI (instr[0]), BY2BI (instr[1]), BY2BI (opcode),          \
           opname == NULL ? "HALT" : opname,                            \
           BY2BI (operand), rand16);                                    \
  } while (0)

#  define DEBUG_INSTRUCTION_LOAD() do {                                 \
    uint8_t *instr = (uint8_t *) &data;                                 \
    int16_t rand16 = tmp->rand;                                         \
    const char *opname = OP_NAME (tmp->rator);                          \
    DEBUG ("      LLL INSTR: " BY2BIPT "" BY2BIPT                       \
           ", RATOR: " BY2BIPT " (%13s), RAND: " BY2BIPT " (%d)",       \
           BY2BI (instr[0]), BY2BI (instr[1]), BY2BI (tmp->rator),      \
           opname == NULL ? "HALT" : opname,                            \
           BY2BI (tmp->rand), rand16);                                  \
  } while (0)

#  define DEBUG_INSTRUCTION_NEXT() do {                                 \
    const char *opname = OP_NAME (pc->rator);                           \
    int16_t rand16 = pc->rand;                                          \
    DEBUG ("     RATOR: " BY2BIPT " (%13s), RAND: " BY2BIPT " (%d)",    \
           BY2BI (pc->rator), opname == NULL ? "HALT" : opname,         \
           BY2BI (pc->rand),  rand16);                                  \
  } while (0)

#  define DEBUG_FAILSTATE() do {                                        \
    DEBUG ("       FAIL[%s]", i);                                       \
    DEBUG ("         NEXT: %s", OP_NAME ((*(pc)).rator));               \
  } while (0)

#  define DEBUG_STACK() do {                                            \
    DEBUG ("         STACK: %p %p", (void *) sp, (void *) m->stack);    \
    for (BacktrackEntry *_tmp_bt = sp; _tmp_bt > m->stack; _tmp_bt--) { \
      DEBUG ("           [I]: %p %p `%s'",                              \
             (void *) _tmp_bt,                                          \
             (void *) (_tmp_bt - 1)->pc,                                \
             (_tmp_bt - 1)->i);                                         \
    }                                                                   \
  } while (0)

# else  /* TEST */
#  define DEBUG(f, ...)
#  define DEBUG_INSTRUCTION()
#  define DEBUG_INSTRUCTION_NEXT()
#  define DEBUG_INSTRUCTION_LOAD()
#  define DEBUG_FAILSTATE()
#  define DEBUG_STACK()
# endif  /* TEST */

char *debug_byte (int8_t data)
{
  const size_t BITS = 8*sizeof(data);
  char *bin_str = malloc (BITS+1);

  for (unsigned int i = 0; i < BITS; i++) {
    unsigned int mask = 1u << (BITS - 1 - i);
    bin_str[i] = (data & mask) ? '1' : '0';
  }
  bin_str[BITS] = '\0';

  return bin_str;
}

#endif  /* DEBUG_H */

/* printf ("FOO: s:%p i:%p i-s:%ld\n", m.s, m.i, m.i - m.s); */

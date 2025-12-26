#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "parser.h"

int main() {
  Parser *p = Parser_New();

  ll_parsing_error err = {0};

  int cursor = 0;

  const char *input = "Identifier <- [a-zA-Z_][a-zA-Z0-9_]*\n";

  Parser_SetInput(p, (const uint8_t *) input, strlen(input));

  ll_tree *t = Parser_Parse(p, &cursor, &err);
  if (!t) {
    fprintf(stderr, "parse error: %s\n", err.message);
    ll_parsing_error_free(&err);
    Parser_Delete(p);
    return 1;
  }

  ll_node_id root;
  if (ll_tree_root(t, &root)) {
    char *pretty = ll_tree_pretty(t, root);
    puts(pretty);
    free(pretty);
  }

  Parser_Delete(p);
  return 0;
}

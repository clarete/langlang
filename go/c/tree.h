#ifndef _LANGLANG_TREE
#define _LANGLANG_TREE

#include <stdbool.h>
#include <stdint.h>

typedef uint32_t ll_node_id;

typedef struct {
  int start, end;
} ll_range;

typedef enum {
  LL_NODE_STRING = 0,
  LL_NODE_SEQUENCE = 1,
  LL_NODE_NODE = 2,
  LL_NODE_ERROR = 3
} ll_node_type;

typedef struct {
  ll_node_type typ;
  int start, end;
  int32_t name_id;
  int32_t child_id;
  int32_t message_id;
} ll_node;

typedef struct {
  int32_t start, end;
} ll_child_range;

typedef struct {
  ll_node *nodes;
  int nodes_len;
  int nodes_cap;

  ll_node_id *children;
  int children_len;
  int children_cap;

  ll_child_range *child_ranges;
  int child_ranges_len;
  int child_ranges_cap;

  const char **strs;
  int strs_len;

  const uint8_t *input;
  int input_len;

  ll_node_id root;
  bool has_root;
} ll_tree;

void ll_tree_init(ll_tree *t);
void ll_tree_free(ll_tree *t);
void ll_tree_destroy(ll_tree *t);
bool ll_tree_root(const ll_tree *t, ll_node_id *out_id);
ll_node_type ll_tree_type(const ll_tree *t, ll_node_id id);
const char *ll_tree_name(const ll_tree *t, ll_node_id id);
ll_range ll_tree_range(const ll_tree *t, ll_node_id id);
bool ll_tree_child(const ll_tree *t, ll_node_id id, ll_node_id *out_child);
int ll_tree_children_len(const ll_tree *t, ll_node_id id);
bool ll_tree_children_at(const ll_tree *t, ll_node_id id, int idx, ll_node_id *out_child);
int ll_tree_children(const ll_tree *t, ll_node_id id, ll_node_id *out, int out_cap);
char *ll_tree_text(const ll_tree *t, ll_node_id id);
char *ll_tree_pretty(const ll_tree *t, ll_node_id id);
char *ll_tree_highlight(const ll_tree *t, ll_node_id id);
ll_tree *ll_tree_copy(const ll_tree *t);

#endif  /*  _LANGLANG_TREE */

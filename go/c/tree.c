#include <stdlib.h>
#include <string.h>
#include <stdio.h>

#ifndef LANGLANG_EMBEDDED
#include "tree.h"
#include "vm.h"
#endif

void *ll_xrealloc(void *p, size_t n);
char *ll_xstrdup(const char *s);

void ll_tree_init(ll_tree *t) {
  memset(t, 0, sizeof(*t));
  t->root = 0;
  t->has_root = false;
}

void ll_tree_free(ll_tree *t) {
  free(t->nodes);
  free(t->children);
  free(t->child_ranges);
  memset(t, 0, sizeof(*t));
}

void ll_tree_reset(ll_tree *t) {
  t->nodes_len = 0;
  t->children_len = 0;
  t->child_ranges_len = 0;
  t->has_root = false;
  t->root = 0;
}

void ll_tree_set_root(ll_tree *t, ll_node_id id) {
  t->root = id;
  t->has_root = true;
}

bool ll_tree_root(const ll_tree *t, ll_node_id *out_id) {
  if (!t->has_root || t->nodes_len == 0) return false;
  *out_id = t->root;
  return true;
}

ll_node_type ll_tree_type(const ll_tree *t, ll_node_id id) {
  return t->nodes[id].typ;
}

const char *ll_tree_name(const ll_tree *t, ll_node_id id) {
  int32_t nid = t->nodes[id].name_id;
  if (nid < 0 || nid >= t->strs_len) return "";
  return t->strs[nid];
}

ll_range ll_tree_range(const ll_tree *t, ll_node_id id) {
  ll_node *n = &t->nodes[id];
  return (ll_range){.start = n->start, .end = n->end};
}

bool ll_tree_child(const ll_tree *t, ll_node_id id, ll_node_id *out_child) {
  const ll_node *n = &t->nodes[id];
  if (n->child_id < 0) return false;
  if (n->typ != LL_NODE_NODE && n->typ != LL_NODE_ERROR) return false;
  *out_child = (ll_node_id)n->child_id;
  return true;
}

int ll_tree_children_len(const ll_tree *t, ll_node_id id) {
  const ll_node *n = &t->nodes[id];
  if (n->typ == LL_NODE_SEQUENCE) {
    if (n->child_id < 0) return 0;
    ll_child_range cr = t->child_ranges[n->child_id];
    return (int)(cr.end - cr.start);
  }
  if (n->typ == LL_NODE_NODE || n->typ == LL_NODE_ERROR) {
    return (n->child_id < 0) ? 0 : 1;
  }
  return 0;
}

bool ll_tree_children_at(const ll_tree *t, ll_node_id id, int idx, ll_node_id *out_child) {
  const ll_node *n = &t->nodes[id];
  if (idx < 0) return false;
  if (n->typ == LL_NODE_SEQUENCE) {
    if (n->child_id < 0) return false;
    ll_child_range cr = t->child_ranges[n->child_id];
    int nchild = (int)(cr.end - cr.start);
    if (idx >= nchild) return false;
    *out_child = t->children[cr.start + idx];
    return true;
  }
  if (n->typ == LL_NODE_NODE || n->typ == LL_NODE_ERROR) {
    if (n->child_id < 0 || idx != 0) return false;
    *out_child = (ll_node_id)n->child_id;
    return true;
  }
  return false;
}

int ll_tree_children(const ll_tree *t, ll_node_id id, ll_node_id *out, int out_cap) {
  int n = ll_tree_children_len(t, id);
  if (!out || out_cap <= 0) return n;
  int m = n < out_cap ? n : out_cap;
  for (int i = 0; i < m; i++) {
    ll_node_id child = 0;
    if (!ll_tree_children_at(t, id, i, &child)) break;
    out[i] = child;
  }
  return n;
}

char *ll_tree_text(const ll_tree *t, ll_node_id id) {
  const ll_node *n = &t->nodes[id];
  switch (n->typ) {
  case LL_NODE_STRING: {
    int start = n->start, end = n->end;
    if (start < 0) start = 0;
    if (end > t->input_len) end = t->input_len;
    if (end < start) end = start;
    int len = end - start;
    char *out = (char *)ll_xrealloc(NULL, (size_t)len + 1);
    memcpy(out, t->input + start, (size_t)len);
    out[len] = 0;
    return out;
  }
  case LL_NODE_SEQUENCE: {
    if (n->child_id < 0) return ll_xstrdup("");
    ll_child_range cr = t->child_ranges[n->child_id];
    // naive O(total) concatenation
    size_t cap = 64, cur = 0;
    char *buf = (char *)ll_xrealloc(NULL, cap);
    buf[0] = 0;
    for (int32_t i = cr.start; i < cr.end; i++) {
      char *part = ll_tree_text(t, t->children[i]);
      size_t plen = strlen(part);
      if (cur + plen + 1 > cap) {
        while (cur + plen + 1 > cap) cap *= 2;
        buf = (char *)ll_xrealloc(buf, cap);
      }
      memcpy(buf + cur, part, plen);
      cur += plen;
      buf[cur] = 0;
      free(part);
    }
    return buf;
  }
  case LL_NODE_NODE:
  case LL_NODE_ERROR: {
    if (n->child_id < 0) {
      if (n->typ == LL_NODE_ERROR) {
        // mimic Go: "error[<label>]"
        const char *label = ll_tree_name(t, id);
        size_t need = strlen("error[]") + strlen(label) + 1;
        char *out = (char *)ll_xrealloc(NULL, need);
        snprintf(out, need, "error[%s]", label);
        return out;
      }
      return ll_xstrdup("");
    }
    return ll_tree_text(t, (ll_node_id)n->child_id);
  }
  default:
    return ll_xstrdup("");
  }
}

static void ll_pretty_ensure(char **out, size_t *len, size_t *cap, size_t add) {
  if (*len + add + 1 <= *cap) return;
  while (*len + add + 1 > *cap) *cap *= 2;
  *out = (char *)ll_xrealloc(*out, *cap);
}

static void ll_pretty_append_n(char **out, size_t *len, size_t *cap, const char *s, size_t n) {
  ll_pretty_ensure(out, len, cap, n);
  memcpy(*out + *len, s, n);
  *len += n;
  (*out)[*len] = 0;
}

static void ll_pretty_append(char **out, size_t *len, size_t *cap, const char *s) {
  ll_pretty_append_n(out, len, cap, s, strlen(s));
}

static void ll_pretty_append_ch(char **out, size_t *len, size_t *cap, char c) {
  ll_pretty_ensure(out, len, cap, 1);
  (*out)[(*len)++] = c;
  (*out)[*len] = 0;
}

static void ll_pretty_append_quoted_slice(char **out, size_t *len, size_t *cap, const uint8_t *buf, int start, int end) {
  ll_pretty_append_ch(out, len, cap, '"');
  for (int i = start; i < end; i++) {
    uint8_t c = buf[i];
    if (c == '\\' || c == '"') {
      ll_pretty_append_ch(out, len, cap, '\\');
      ll_pretty_append_ch(out, len, cap, (char)c);
    } else if (c == '\n') {
      ll_pretty_append_ch(out, len, cap, '\\');
      ll_pretty_append_ch(out, len, cap, 'n');
    } else if (c == '\r') {
      ll_pretty_append_ch(out, len, cap, '\\');
      ll_pretty_append_ch(out, len, cap, 'r');
    } else if (c == '\t') {
      ll_pretty_append_ch(out, len, cap, '\\');
      ll_pretty_append_ch(out, len, cap, 't');
    } else {
      ll_pretty_append_ch(out, len, cap, (char)c);
    }
  }
  ll_pretty_append_ch(out, len, cap, '"');
}

static void ll_tree_pretty_rec(const ll_tree *t,
                               ll_node_id id,
                               char **out,
                               size_t *len,
                               size_t *cap,
                               const char *prefix,
                               bool is_last,
                               bool is_root) {
  const ll_node *n = &t->nodes[id];

  // Prefix + branch connectors (Unicode box-drawing, like the Go pretty-printer).
  if (prefix && prefix[0]) ll_pretty_append(out, len, cap, prefix);
  if (!is_root) {
    ll_pretty_append(out, len, cap, is_last ? "└── " : "├── ");
  }

  // Render node label.
  if (n->typ == LL_NODE_STRING) {
    int start = n->start, end = n->end;
    start = start < 0 ? 0 : start;
    end = end > t->input_len ? t->input_len : end;
    if (end < start) end = start;
    ll_pretty_append_quoted_slice(out, len, cap, t->input, start, end);
    ll_pretty_append_ch(out, len, cap, '\n');
    return;
  }

  if (n->typ == LL_NODE_SEQUENCE) {
    char buf[64];
    int wrote = snprintf(buf, sizeof(buf), "Sequence (%d..%d)", n->start, n->end);
    if (wrote < 0) wrote = 0;
    ll_pretty_append_n(out, len, cap, buf, (size_t)wrote);
    ll_pretty_append_ch(out, len, cap, '\n');
  } else if (n->typ == LL_NODE_NODE) {
    const char *name = ll_tree_name(t, id);
    // write: "<name> (start..end)\n"
    ll_pretty_append(out, len, cap, name);
    char buf[48];
    int wrote = snprintf(buf, sizeof(buf), " (%d..%d)", n->start, n->end);
    if (wrote < 0) wrote = 0;
    ll_pretty_append_n(out, len, cap, buf, (size_t)wrote);
    ll_pretty_append_ch(out, len, cap, '\n');
  } else if (n->typ == LL_NODE_ERROR) {
    const char *label = ll_tree_name(t, id);
    ll_pretty_append(out, len, cap, "Error<");
    ll_pretty_append(out, len, cap, label);
    ll_pretty_append(out, len, cap, ">");
    char buf[48];
    int wrote = snprintf(buf, sizeof(buf), " (%d..%d)", n->start, n->end);
    if (wrote < 0) wrote = 0;
    ll_pretty_append_n(out, len, cap, buf, (size_t)wrote);
    ll_pretty_append_ch(out, len, cap, '\n');
  } else {
    ll_pretty_append(out, len, cap, "(unknown)\n");
    return;
  }

  // Recurse
  const char *pad = "";
  if (!is_root) pad = is_last ? "    " : "│   ";

  size_t plen = prefix ? strlen(prefix) : 0;
  size_t padlen = strlen(pad);
  char *next_prefix = NULL;
  if (plen + padlen > 0) {
    next_prefix = (char *)ll_xrealloc(NULL, plen + padlen + 1);
    if (plen) memcpy(next_prefix, prefix, plen);
    if (padlen) memcpy(next_prefix + plen, pad, padlen);
    next_prefix[plen + padlen] = 0;
  }

  const char *np = next_prefix ? next_prefix : "";

  if (n->typ == LL_NODE_SEQUENCE && n->child_id >= 0) {
    ll_child_range cr = t->child_ranges[n->child_id];
    for (int32_t i = cr.start; i < cr.end; i++) {
      bool child_last = (i == cr.end - 1);
      ll_tree_pretty_rec(t, t->children[i], out, len, cap, np, child_last, false);
    }
  } else if ((n->typ == LL_NODE_NODE || n->typ == LL_NODE_ERROR) && n->child_id >= 0) {
    ll_tree_pretty_rec(t, (ll_node_id)n->child_id, out, len, cap, np, true, false);
  }

  free(next_prefix);
}

char *ll_tree_pretty(const ll_tree *t, ll_node_id id) {
  size_t cap = 1024, len = 0;
  char *out = (char *)ll_xrealloc(NULL, cap);
  out[0] = 0;
  ll_tree_pretty_rec(t, id, &out, &len, &cap, "", true, true);
  out[len] = 0;
  return out;
}

char *ll_tree_highlight(const ll_tree *t, ll_node_id id) {
  // TODO: add ANSI colors (like Go's tree.Highlight). For now, Pretty() already
  // prints the full structure with Unicode connectors.
  return ll_tree_pretty(t, id);
}

ll_tree *ll_tree_copy(const ll_tree *t) {
  if (!t) return NULL;
  ll_tree *out = (ll_tree *)ll_xrealloc(NULL, sizeof(ll_tree));
  memset(out, 0, sizeof(*out));

  out->nodes_len = t->nodes_len;
  out->nodes_cap = t->nodes_len;
  if (t->nodes_len > 0) {
    out->nodes = (ll_node *)ll_xrealloc(NULL, (size_t)t->nodes_len * sizeof(ll_node));
    memcpy(out->nodes, t->nodes, (size_t)t->nodes_len * sizeof(ll_node));
  }

  out->children_len = t->children_len;
  out->children_cap = t->children_len;
  if (t->children_len > 0) {
    out->children = (ll_node_id *)ll_xrealloc(NULL, (size_t)t->children_len * sizeof(ll_node_id));
    memcpy(out->children, t->children, (size_t)t->children_len * sizeof(ll_node_id));
  }

  out->child_ranges_len = t->child_ranges_len;
  out->child_ranges_cap = t->child_ranges_len;
  if (t->child_ranges_len > 0) {
    out->child_ranges = (ll_child_range *)ll_xrealloc(NULL, (size_t)t->child_ranges_len * sizeof(ll_child_range));
    memcpy(out->child_ranges, t->child_ranges, (size_t)t->child_ranges_len * sizeof(ll_child_range));
  }

  // Borrowed fields (match Go Tree.Copy docs)
  out->strs = t->strs;
  out->strs_len = t->strs_len;
  out->input = t->input;
  out->input_len = t->input_len;

  out->root = t->root;
  out->has_root = t->has_root;
  return out;
}

void ll_tree_destroy(ll_tree *t) {
  if (!t) return;
  ll_tree_free(t);
  free(t);
}

/* internal utilities */

void ll_tree_bind_input(ll_tree *t, const uint8_t *input, int input_len) {
  t->input = input;
  t->input_len = input_len;
}

void ll_tree_bind_strings(ll_tree *t, const char **strs, int strs_len) {
  t->strs = strs;
  t->strs_len = strs_len;
}

static void _ll_tree_grow_nodes(ll_tree *t, int need) {
  if (t->nodes_len + need <= t->nodes_cap) return;
  int cap = t->nodes_cap ? t->nodes_cap : 256;
  while (cap < t->nodes_len + need) cap *= 2;
  t->nodes = (ll_node *)ll_xrealloc(t->nodes, (size_t)cap * sizeof(ll_node));
  t->nodes_cap = cap;
}

static void _ll_tree_grow_children(ll_tree *t, int need) {
  if (t->children_len + need <= t->children_cap) return;
  int cap = t->children_cap ? t->children_cap : 512;
  while (cap < t->children_len + need) cap *= 2;
  t->children = (ll_node_id *)ll_xrealloc(t->children, (size_t)cap * sizeof(ll_node_id));
  t->children_cap = cap;
}

static void _ll_tree_grow_child_ranges(ll_tree *t, int need) {
  if (t->child_ranges_len + need <= t->child_ranges_cap) return;
  int cap = t->child_ranges_cap ? t->child_ranges_cap : 256;
  while (cap < t->child_ranges_len + need) cap *= 2;
  t->child_ranges = (ll_child_range *)ll_xrealloc(t->child_ranges, (size_t)cap * sizeof(ll_child_range));
  t->child_ranges_cap = cap;
}

/* utilities used by the VM */

ll_node_id ll_tree_add_string(ll_tree *t, int start, int end) {
  _ll_tree_grow_nodes(t, 1);
  ll_node_id id = (ll_node_id)t->nodes_len;
  t->nodes[t->nodes_len++] = (ll_node){
    .typ = LL_NODE_STRING,
    .start = start,
    .end = end,
    .name_id = -1,
    .child_id = -1,
    .message_id = -1,
  };
  return id;
}

ll_node_id ll_tree_add_sequence(ll_tree *t, const ll_node_id *children, int children_len, int start, int end) {
  _ll_tree_grow_nodes(t, 1);
  int32_t child_range_id = -1;
  if (children_len > 0) {
    _ll_tree_grow_child_ranges(t, 1);
    child_range_id = (int32_t)t->child_ranges_len;
    int32_t child_start = (int32_t)t->children_len;
    _ll_tree_grow_children(t, children_len);
    memcpy(&t->children[t->children_len], children, (size_t)children_len * sizeof(ll_node_id));
    t->children_len += children_len;
    int32_t child_end = (int32_t)t->children_len;
    t->child_ranges[t->child_ranges_len++] = (ll_child_range){.start = child_start, .end = child_end};
  }
  ll_node_id id = (ll_node_id)t->nodes_len;
  t->nodes[t->nodes_len++] = (ll_node){
    .typ = LL_NODE_SEQUENCE,
    .start = start,
    .end = end,
    .name_id = -1,
    .child_id = child_range_id,
    .message_id = -1,
  };
  return id;
}

ll_node_id ll_tree_add_node(ll_tree *t, int32_t name_id, ll_node_id child, int start, int end) {
  _ll_tree_grow_nodes(t, 1);
  ll_node_id id = (ll_node_id)t->nodes_len;
  t->nodes[t->nodes_len++] = (ll_node){
    .typ = LL_NODE_NODE,
    .start = start,
    .end = end,
    .name_id = name_id,
    .child_id = (int32_t)child,
    .message_id = -1,
  };
  return id;
}

ll_node_id ll_tree_add_error(ll_tree *t,
                             int32_t label_id,
                             int32_t message_id,
                             int start,
                             int end)
{
  _ll_tree_grow_nodes(t, 1);
  ll_node_id id = (ll_node_id)t->nodes_len;
  t->nodes[t->nodes_len++] = (ll_node){
    .typ = LL_NODE_ERROR,
    .start = start,
    .end = end,
    .name_id = label_id,
    .child_id = -1,
    .message_id = message_id,
  };
  return id;
}

ll_node_id ll_tree_add_error_with_child(ll_tree *t,
                                        int32_t label_id,
                                        int32_t message_id,
                                        ll_node_id child_id,
                                        int start,
                                        int end)
{
  _ll_tree_grow_nodes(t, 1);
  ll_node_id id = (ll_node_id)t->nodes_len;
  t->nodes[t->nodes_len++] = (ll_node){
    .typ = LL_NODE_ERROR,
    .start = start,
    .end = end,
    .name_id = label_id,
    .child_id = (int32_t)child_id,
    .message_id = message_id,
  };
  return id;
}

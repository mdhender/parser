#!/usr/bin/env bash
# Explicit C/Go behavioral comparison, separate from offline Go tests.
set -euo pipefail
export LC_ALL=C TZ=UTC GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local CGO_ENABLED=0
root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
compiler="${CC:-cc}"
for tool in "$compiler" go mktemp cat diff rm; do
  if ! command -v "$tool" >/dev/null; then
    echo "Required Go runtime comparison tool not found: $tool" >&2; exit 2
  fi
done
work="$(mktemp -d "${TMPDIR:-/tmp}/lemon-go-runtime.XXXXXXXX")"
cleanup() {
  if (( $? == 0 )); then rm -rf "$work"; else
    echo "Runtime comparison failed; artifacts retained at $work" >&2
    printf 'Reproduce: %q\n' "$root/scripts/test-go-runtime.sh" >&2
  fi
}
trap cleanup EXIT
"$compiler" -o "$work/lemon" "$root/reference/sqlite/tool/lemon.c"
(cd "$root" && go build -o "$work/glcc" ./cmd/glcc && go build -o "$work/glemon" ./cmd/glemon)
cat >"$work/c.y" <<'GRAMMAR'
%extra_argument {int *out}
%extra_context {int *stats}
%include {
#include <stdio.h>
#include <stdlib.h>
}
GRAMMAR
cat >"$work/g.y" <<'GRAMMAR'
%extra_argument {out *int}
%extra_context {stats *int}
GRAMMAR
cat "$root/testdata/go/runtime.y" >>"$work/c.y"
cat "$root/testdata/go/runtime.y" >>"$work/g.y"
cat >>"$work/c.y" <<'GRAMMAR'
%code {
int main(void) {
  int result = 0, stats = 0;
  void *p = RuntimeAlloc(malloc, &stats);
  int tokens[][2] = {{TK_NUM,2},{TK_PLUS,0},{TK_NUM,3},{TK_TIMES,0},
    {TK_LP,0},{TK_NUM,4},{TK_MINUS,0},{TK_NUM,1},{TK_RP,0},{TK_SEMI,0},{0,0},
    {TK_KEY,3},{TK_SEMI,0},{TK_OTHER,7},{TK_SEMI,0},{0,0},
    {TK_NUM,5},{TK_SEMI,0},{TK_NUM,0},{TK_LP,7},{TK_SEMI,0},
    {TK_NUM,11},{TK_SEMI,0},{0,0}};
  for (unsigned i=0; i<sizeof(tokens)/sizeof(tokens[0]); i++) {
    Runtime(p,tokens[i][0],tokens[i][1],&result);
    if (tokens[i][0]==0) printf("%d %d\n",result,stats);
  }
  Runtime(p,TK_NUM,13,&result); RuntimeFinalize(p);
  printf("%d %d\n",result,stats);
  RuntimeInit(p,&stats); Runtime(p,TK_LP,0,&result); Runtime(p,0,0,&result);
  printf("%d %d\n",result,stats);
  Runtime(p,TK_LP,0,&result); Runtime(p,0,0,&result);
  printf("%d %d\n",result,stats);
  RuntimeFree(p,free); return 0;
}
}
GRAMMAR
cat >"$work/main.go" <<'GO'
package main
import "fmt"
func main() {
  result, stats := 0, 0
  p := NewRuntime(&stats)
  tokens := [][2]int{{TK_NUM,2},{TK_PLUS,0},{TK_NUM,3},{TK_TIMES,0},
    {TK_LP,0},{TK_NUM,4},{TK_MINUS,0},{TK_NUM,1},{TK_RP,0},{TK_SEMI,0},{0,0},
    {TK_KEY,3},{TK_SEMI,0},{TK_OTHER,7},{TK_SEMI,0},{0,0},
    {TK_NUM,5},{TK_SEMI,0},{TK_NUM,0},{TK_LP,7},{TK_SEMI,0},
    {TK_NUM,11},{TK_SEMI,0},{0,0}}
  for _, token := range tokens {
    p.Parse(token[0],token[1],&result)
    if token[0]==0 { fmt.Printf("%d %d\n",result,stats) }
  }
  p.Parse(TK_NUM,13,&result); p.Close()
  fmt.Printf("%d %d\n",result,stats)
  p.Parse(TK_LP,0,&result); p.Parse(0,0,&result)
  fmt.Printf("%d %d\n",result,stats)
  p.Parse(TK_LP,0,&result); p.Parse(0,0,&result)
  fmt.Printf("%d %d\n",result,stats)
  p.Close()
}
GO
for mode in compressed uncompressed; do
  flags=()
  if [[ "$mode" == uncompressed ]]; then flags+=(-c); fi
  "$work/lemon" -q "${flags[@]}" -T"$root/reference/sqlite/tool/lempar.c" "$work/c.y"
  "$compiler" -o "$work/c-parser" "$work/c.c"
  "$work/c-parser" >"$work/c-$mode.txt"
  for command in glcc glemon; do
    "$work/$command" -q "${flags[@]}" -package main "$work/g.y"
    go run "$work/main.go" "$work/g.go" >"$work/$command-$mode.txt"
    diff -u "$work/c-$mode.txt" "$work/$command-$mode.txt"
  done
done
echo 'PASS: C/Go runtime behavior matches for both commands, compressed and uncompressed'

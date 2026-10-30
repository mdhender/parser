//go:build ignore

/* Milestone B instrumentation only. Never use this for parser artifacts.
** Build: cc -o lemon reference/grammar-snapshot.c
** Run: ./lemon [-DNAME[=VALUE]] [-UNAME] [-l] grammar.y
** The pinned implementation is included unchanged. The initialization and
** indexing below intentionally follow its main(), stopping before FIRST sets.
*/
#define main lemon_main
#include "sqlite/tool/lemon.c"
#undef main

/* A NULL string is distinct from an allocated empty string. Encode bytes,
** not characters; this also makes tabs, newlines and non-ASCII unambiguous. */
static void snapshot_string(const char *z){
  const unsigned char *p = (const unsigned char *)z;
  if( z==0 ){
    putchar('-');
  }else{
    for(; *p; p++) printf("%02x", (unsigned int)*p);
  }
}

static void snapshot_symbol_name(const struct symbol *sp){
  snapshot_string(sp ? sp->name : 0);
}

static void snapshot_constituents(const struct symbol *sp){
  int i;
  for(i=0; i<sp->nsubsym; i++){
    if( i ) putchar(',');
    snapshot_symbol_name(sp->subsym[i]);
  }
}

static void snapshot_decl(const char *keyword, const char *value){
  printf("decl\t%s\t", keyword);
  snapshot_string(value);
  putchar('\n');
}

static void snapshot_grammar(struct lemon *lem, int symbolCount){
  int i, j;
  struct rule *rp;
  printf("grammar\t%d\t%d\t%d\t%d\t", lem->nsymbol,
         lem->nterminal, lem->nruleWithAction, lem->has_fallback);
  snapshot_symbol_name(lem->wildcard);
  printf("\t%d\n", lem->startRule ? lem->startRule->index : -1);
  snapshot_decl("name", lem->name);
  snapshot_decl("include", lem->include);
  snapshot_decl("code", lem->extracode);
  snapshot_decl("token_destructor", lem->tokendest);
  snapshot_decl("default_destructor", lem->vardest);
  snapshot_decl("token_prefix", lem->tokenprefix);
  snapshot_decl("syntax_error", lem->error);
  snapshot_decl("parse_accept", lem->accept);
  snapshot_decl("parse_failure", lem->failure);
  snapshot_decl("stack_overflow", lem->overflow);
  snapshot_decl("extra_argument", lem->arg);
  snapshot_decl("extra_context", lem->ctx);
  snapshot_decl("token_type", lem->tokentype);
  snapshot_decl("default_type", lem->vartype);
  snapshot_decl("realloc", lem->reallocFunc);
  snapshot_decl("free", lem->freeFunc);
  snapshot_decl("stack_size", lem->stacksize);
  snapshot_decl("start_symbol", lem->start);
  for(i=0; i<symbolCount; i++){
    struct symbol *sp = lem->symbols[i];
    printf("symbol\t");
    snapshot_symbol_name(sp);
    printf("\t%d\t%d\t%d\t%d\t%d\t", sp->index, (int)sp->type,
           sp->useCnt, sp->prec, (int)sp->assoc);
    snapshot_symbol_name(sp->fallback);
    putchar('\t');
    snapshot_string(sp->destructor);
    printf("\t%d\t", sp->destLineno);
    snapshot_string(sp->datatype);
    printf("\t%d\t", sp->bContent);
    snapshot_constituents(sp);
    putchar('\t');
    for(rp=sp->rule; rp; rp=rp->nextlhs){
      if( rp!=sp->rule ) putchar(',');
      printf("%d", rp->index);
    }
    putchar('\n');
  }
  for(rp=lem->rule; rp; rp=rp->next){
    printf("rule\t%d\t%d\t", rp->index, rp->iRule);
    snapshot_symbol_name(rp->lhs);
    putchar('\t');
    snapshot_string(rp->lhsalias);
    printf("\t%d\t%d\t", rp->ruleline, rp->line);
    snapshot_string(rp->code);
    printf("\t%d\t%d\t", (int)rp->noCode, (int)rp->neverReduce);
    snapshot_symbol_name(rp->precsym);
    printf("\t%d\n", rp->nrhs);
    for(j=0; j<rp->nrhs; j++){
      printf("rhs\t");
      snapshot_symbol_name(rp->rhs[j]);
      printf("\t%d\t", (int)rp->rhs[j]->type);
      snapshot_string(rp->rhsalias[j]);
      putchar('\t');
      snapshot_constituents(rp->rhs[j]);
      putchar('\n');
    }
  }
  for(i=0; i<nDefine; i++){
    printf("define\t");
    snapshot_string(azDefine[i]);
    printf("\t%d\n", (int)bDefineUsed[i]);
  }
}

int main(int argc, char **argv){
  static int nolinenosflag = 0;
  static struct s_options options[] = {
    {OPT_FSTR, "D", (char*)handle_D_option, "Define an %ifdef macro."},
    {OPT_FLAG, "l", (char*)&nolinenosflag, "Do not print #line statements."},
    {OPT_FSTR, "U", (char*)handle_U_option, "Undefine a macro."},
    {OPT_FLAG, 0, 0, 0}
  };
  struct lemon lem;
  struct rule *rp;
  int i, symbolCount, status;
  OptInit(argv, options, stderr);
  if( OptNArgs()!=1 ){
    fprintf(stderr, "Exactly one filename argument is required.\n");
    return 1;
  }
  memset(&lem, 0, sizeof(lem));
  qsort(azDefine, nDefine, sizeof(azDefine[0]), defineCmp);
  Strsafe_init();
  Symbol_init();
  State_init();
  lem.argv = argv;
  lem.argc = argc;
  lem.filename = OptArg(0);
  lem.nolinenosflag = nolinenosflag;
  Symbol_new("$");
  Parse(&lem);
  if( lem.errorcnt ){
    status = lem.errorcnt;
    lemon_free_all();
    return status;
  }
  if( lem.nrule==0 ){
    fprintf(stderr, "Empty grammar.\n");
    lemon_free_all();
    return 1;
  }
  lem.errsym = Symbol_find("error");
  Symbol_new("{default}");
  symbolCount = lem.nsymbol = Symbol_count();
  lem.symbols = Symbol_arrayof();
  for(i=0; i<lem.nsymbol; i++) lem.symbols[i]->index = i;
  qsort(lem.symbols, lem.nsymbol, sizeof(struct symbol*), Symbolcmpp);
  for(i=0; i<lem.nsymbol; i++) lem.symbols[i]->index = i;
  while( lem.symbols[i-1]->type==MULTITERMINAL ){ i--; }
  assert( strcmp(lem.symbols[i-1]->name, "{default}")==0 );
  lem.nsymbol = i - 1;
  for(i=1; ISUPPER(lem.symbols[i]->name[0]); i++);
  lem.nterminal = i;
  for(i=0, rp=lem.rule; rp; rp=rp->next){
    rp->iRule = rp->code ? i++ : -1;
  }
  lem.nruleWithAction = i;
  for(rp=lem.rule; rp; rp=rp->next){
    if( rp->iRule<0 ) rp->iRule = i++;
  }
  lem.startRule = lem.rule;
  lem.rule = Rule_sort(lem.rule);
  FindRulePrecedences(&lem);
  snapshot_grammar(&lem, symbolCount);
  lemon_free_all();
  return 0;
}

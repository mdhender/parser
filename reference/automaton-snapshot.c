//go:build ignore

/* Milestone C instrumentation only. Never use this for parser artifacts.
** Build: cc -o lemon reference/automaton-snapshot.c
** Run: ./lemon [-DNAME[=VALUE]] [-UNAME] [-l] grammar.y
** Include the pinned source unchanged; mirror grammar-snapshot.c initialization
** and main() through FindActions, before compression or state resorting.
*/
#define main lemon_main
#include "sqlite/tool/lemon.c"
#undef main

static void snapshot_set(const char *set, int limit){
  int i, comma = 0;
  if( set ){
    for(i=0; i<limit; i++){
      if( SetFind(set,i) ){
        if( comma ) putchar(',');
        printf("%d", i);
        comma = 1;
      }
    }
  }
}

static void snapshot_automaton(struct lemon *lem){
  int i;
  struct rule *rp;
  printf("automaton\t%d\t%d\t%d\t%d\t%d\n", lem->nsymbol,
         lem->nterminal, lem->nrule, lem->nstate, lem->nconflict);
  for(i=0; i<lem->nsymbol; i++){
    struct symbol *sp = lem->symbols[i];
    printf("first\t%d\t%d\t", sp->index, (int)sp->lambda);
    snapshot_set(sp->firstset, lem->nterminal);
    putchar('\n');
  }
  for(rp=lem->rule; rp; rp=rp->next){
    printf("reduce\t%d\t%d\t%d\n", rp->index, rp->lhsStart,
           (int)rp->canReduce);
  }
  for(i=0; i<lem->nstate; i++){
    struct state *stp = lem->sorted[i];
    struct config *cfp;
    struct action *ap;
    printf("state\t%d\n", stp->statenum);
    for(cfp=stp->bp; cfp; cfp=cfp->bp){
      printf("basis\t%d\t%d\n", cfp->rp->index, cfp->dot);
    }
    for(cfp=stp->cfp; cfp; cfp=cfp->next){
      printf("config\t%d\t%d\t", cfp->rp->index, cfp->dot);
      snapshot_set(cfp->fws, lem->nterminal+1);
      putchar('\n');
    }
    /* Do not sort again: resolution changes types in the sorted list. */
    for(ap=stp->ap; ap; ap=ap->next){
      printf("action\t%d\t%d\t", ap->sp->index, (int)ap->type);
      switch( ap->type ){
        case SHIFT:
        case SH_RESOLVED:
        case SSCONFLICT:
          printf("%d", ap->x.stp->statenum);
          break;
        case REDUCE:
        case SRCONFLICT:
        case RRCONFLICT:
        case RD_RESOLVED:
          printf("%d", ap->x.rp->index);
          break;
        default:
          /* In particular ERROR may still contain the old SHIFT pointer. */
          putchar('-');
          break;
      }
      putchar('\n');
    }
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
  int i, status;
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
  lem.nsymbol = Symbol_count();
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
  SetSize(lem.nterminal+1);
  FindRulePrecedences(&lem);
  FindFirstSets(&lem);
  lem.nstate = 0;
  FindStates(&lem);
  lem.sorted = State_arrayof();
  FindLinks(&lem);
  FindFollowSets(&lem);
  FindActions(&lem);
  snapshot_automaton(&lem);
  if( lem.nconflict>0 ){
    fprintf(stderr, "%d parsing conflicts.\n", lem.nconflict);
  }
  status = (lem.errorcnt>0 || lem.nconflict>0) ? 1 : 0;
  lemon_free_all();
  return status;
}

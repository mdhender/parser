//go:build ignore

/* Milestone D instrumentation only. Never use this for parser artifacts.
** Build: cc -o lemon reference/table-snapshot.c
** Run: ./lemon [-c] [-r] [-DNAME[=VALUE]] [-UNAME] [-l] grammar.y
** Include the pinned source unchanged. Follow main() through CompressTables
** and ResortStates, then run only ReportTable's packing and doesReduce blocks.
** No template, translate_code, or emitter processing is performed.
*/
#define main lemon_main
#include "sqlite/tool/lemon.c"
#undef main

static void snapshot_tables(struct lemon *lemp){
  struct state *stp;
  struct action *ap;
  struct rule *rp;
  struct acttab *pActtab;
  int i, n;
  int mnTknOfst, mxTknOfst;
  int mnNtOfst, mxNtOfst;
  struct axset *ax;

  lemp->minShiftReduce = lemp->nstate;
  lemp->errAction = lemp->minShiftReduce + lemp->nrule;
  lemp->accAction = lemp->errAction + 1;
  lemp->noAction = lemp->accAction + 1;
  lemp->minReduce = lemp->noAction + 1;
  lemp->maxAction = lemp->minReduce + lemp->nrule;

  /* Verbatim ReportTable axset/acttab packing and doesReduce blocks. */
  /* Compute the action table, but do not output it yet.  The action
  ** table must be computed before generating the YYNSTATE macro because
  ** we need to know how many states can be eliminated.
  */
  ax = (struct axset *) lemon_calloc(lemp->nxstate*2, sizeof(ax[0]));
  if( ax==0 ){
    fprintf(stderr,"malloc failed\n");
    exit(1);
  }
  for(i=0; i<lemp->nxstate; i++){
    stp = lemp->sorted[i];
    ax[i*2].stp = stp;
    ax[i*2].isTkn = 1;
    ax[i*2].nAction = stp->nTknAct;
    ax[i*2+1].stp = stp;
    ax[i*2+1].isTkn = 0;
    ax[i*2+1].nAction = stp->nNtAct;
  }
  mxTknOfst = mnTknOfst = 0;
  mxNtOfst = mnNtOfst = 0;
  /* In an effort to minimize the action table size, use the heuristic
  ** of placing the largest action sets first */
  for(i=0; i<lemp->nxstate*2; i++) ax[i].iOrder = i;
  qsort(ax, lemp->nxstate*2, sizeof(ax[0]), axset_compare);
  pActtab = acttab_alloc(lemp->nsymbol, lemp->nterminal);
  for(i=0; i<lemp->nxstate*2 && ax[i].nAction>0; i++){
    stp = ax[i].stp;
    if( ax[i].isTkn ){
      for(ap=stp->ap; ap; ap=ap->next){
        int action;
        if( ap->sp->index>=lemp->nterminal ) continue;
        action = compute_action(lemp, ap);
        if( action<0 ) continue;
        acttab_action(pActtab, ap->sp->index, action);
      }
      stp->iTknOfst = acttab_insert(pActtab, 1);
      if( stp->iTknOfst<mnTknOfst ) mnTknOfst = stp->iTknOfst;
      if( stp->iTknOfst>mxTknOfst ) mxTknOfst = stp->iTknOfst;
    }else{
      for(ap=stp->ap; ap; ap=ap->next){
        int action;
        if( ap->sp->index<lemp->nterminal ) continue;
        if( ap->sp->index==lemp->nsymbol ) continue;
        action = compute_action(lemp, ap);
        if( action<0 ) continue;
        acttab_action(pActtab, ap->sp->index, action);
      }
      stp->iNtOfst = acttab_insert(pActtab, 0);
      if( stp->iNtOfst<mnNtOfst ) mnNtOfst = stp->iNtOfst;
      if( stp->iNtOfst>mxNtOfst ) mxNtOfst = stp->iNtOfst;
    }
#if 0  /* Uncomment for a trace of how the yy_action[] table fills out */
    { int jj, nn;
      for(jj=nn=0; jj<pActtab->nAction; jj++){
        if( pActtab->aAction[jj].action<0 ) nn++;
      }
      printf("%4d: State %3d %s n: %2d size: %5d freespace: %d\n",
             i, stp->statenum, ax[i].isTkn ? "Token" : "Var  ",
             ax[i].nAction, pActtab->nAction, nn);
    }
#endif
  }
  lemon_free(ax);

  /* Mark rules that are actually used for reduce actions after all
  ** optimizations have been applied
  */
  for(rp=lemp->rule; rp; rp=rp->next) rp->doesReduce = LEMON_FALSE;
  for(i=0; i<lemp->nxstate; i++){
    for(ap=lemp->sorted[i]->ap; ap; ap=ap->next){
      if( ap->type==REDUCE || ap->type==SHIFTREDUCE ){
        ap->x.rp->doesReduce = 1;
      }
    }
  }

  printf("tables\t%d\t%d\t%d\t%d\t%d\t%d\n", lemp->nsymbol,
         lemp->nterminal, lemp->nrule, lemp->nstate, lemp->nxstate,
         lemp->nconflict);
  printf("codes\t%d\t%d\t%d\t%d\t%d\t%d\n", lemp->minShiftReduce,
         lemp->errAction, lemp->accAction, lemp->noAction,
         lemp->minReduce, lemp->maxAction);
  printf("bounds\t%d\t%d\t%d\t%d\n", mnTknOfst, mxTknOfst,
         mnNtOfst, mxNtOfst);
  printf("sizes\t%d\t%d\n", acttab_action_size(pActtab),
         acttab_lookahead_size(pActtab));
  for(rp=lemp->rule; rp; rp=rp->next){
    printf("rule\t%d\t%d\n", rp->index, (int)rp->doesReduce);
  }
  for(i=0; i<lemp->nstate; i++){
    stp = lemp->sorted[i];
    printf("state\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n",
           stp->statenum, stp->nTknAct, stp->nNtAct, stp->iTknOfst,
           stp->iNtOfst, stp->iDfltReduce, (int)stp->autoReduce,
           stp->pDfltReduce ? stp->pDfltReduce->index : -1);
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
        case NOT_USED:
        case SHIFTREDUCE:
          printf("%d", ap->x.rp->index);
          break;
        default:
          putchar('-');
          break;
      }
      printf("\t%d\t%d\n", ap->spOpt ? ap->spOpt->index : -1,
             compute_action(lemp, ap));
    }
  }
  for(i=0; i<pActtab->nAction; i++){
    printf("entry\t%d\t%d\t%d\n", i, acttab_yylookahead(pActtab, i),
           acttab_yyaction(pActtab, i));
  }

  /* ReportTable's final array values, without C formatting or byte sizes. */
  lemp->nactiontab = n = acttab_action_size(pActtab);
  for(i=0; i<n; i++){
    int action = acttab_yyaction(pActtab, i);
    if( action<0 ) action = lemp->noAction;
    printf("yy-action\t%d\t%d\n", i, action);
  }
  lemp->nlookaheadtab = n = acttab_lookahead_size(pActtab);
  for(i=0; i<n; i++){
    int la = acttab_yylookahead(pActtab, i);
    if( la<0 ) la = lemp->nsymbol;
    printf("yy-lookahead\t%d\t%d\n", i, la);
  }
  while( i<lemp->nterminal+lemp->nactiontab ){
    printf("yy-lookahead\t%d\t%d\n", i++, lemp->nterminal);
  }
  n = lemp->nxstate;
  while( n>0 && lemp->sorted[n-1]->iTknOfst==NO_OFFSET ) n--;
  for(i=0; i<n; i++){
    int ofst = lemp->sorted[i]->iTknOfst;
    if( ofst==NO_OFFSET ) ofst = lemp->nactiontab;
    printf("yy-shift\t%d\t%d\n", i, ofst);
  }
  n = lemp->nxstate;
  while( n>0 && lemp->sorted[n-1]->iNtOfst==NO_OFFSET ) n--;
  for(i=0; i<n; i++){
    int ofst = lemp->sorted[i]->iNtOfst;
    if( ofst==NO_OFFSET ) ofst = mnNtOfst - 1;
    printf("yy-reduce\t%d\t%d\n", i, ofst);
  }
  for(i=0; i<lemp->nxstate; i++){
    int action = lemp->sorted[i]->iDfltReduce;
    action = action<0 ? lemp->errAction : lemp->minReduce+action;
    printf("yy-default\t%d\t%d\n", i, action);
  }
  acttab_free(pActtab);
}

int main(int argc, char **argv){
  static int nolinenosflag = 0;
  static int compress = 0;
  static int noResort = 0;
  static struct s_options options[] = {
    {OPT_FLAG, "c", (char*)&compress, "Don't compress the action table."},
    {OPT_FSTR, "D", (char*)handle_D_option, "Define an %ifdef macro."},
    {OPT_FLAG, "l", (char*)&nolinenosflag, "Do not print #line statements."},
    {OPT_FLAG, "r", (char*)&noResort, "Do not sort or renumber states"},
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
  if( compress==0 ) CompressTables(&lem);
  if( noResort==0 ) ResortStates(&lem);
  snapshot_tables(&lem);
  if( lem.nconflict>0 ){
    fprintf(stderr, "%d parsing conflicts.\n", lem.nconflict);
  }
  status = (lem.errorcnt>0 || lem.nconflict>0) ? 1 : 0;
  lemon_free_all();
  return status;
}

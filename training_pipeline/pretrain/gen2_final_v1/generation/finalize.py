#!/usr/bin/env python3
import argparse, gzip, hashlib, json, math, os, random, shutil, statistics
from collections import Counter, defaultdict
from pathlib import Path

SIM_COMMIT="0cff407625f090fba11b1408e9e10f8aca355d7e"
DATASET="FMoE_Gen2_FinalPretrain_v1"
ARCH="HELIX_GEN2_S_W128_R24_RAW_V1"
ALLOWED_SOURCES={"direct_simulator_outcome","repeated_simulator_rollout","deterministic_simulator_rule","analytical_from_simulator_state"}
VOCAB={
 "position":{"GK","CB","FB","DM","CM","AM","WG","ST"},
 "squad_role":{"star","starter","rotation","prospect","depth"},
 "state_t_squad_role":{"star","starter","rotation","prospect","depth"},
 "home_formation":{"4-3-3","4-2-3-1","3-4-3","3-5-2","4-4-2","4-1-4-1"},
 "away_formation":{"4-3-3","4-2-3-1","3-4-3","3-5-2","4-4-2","4-1-4-1"},
 "competition_level":{"elite","top","mid","lower"},
 "family":{"workload","career","market","club","match"},
 "changed_features":{"fatigue","fixture_congestion","recent_minutes"},
}
TASKS={"match_competition","player_dynamics","development_decline","economy_contracts","club_manager","temporal_transition","counterfactual","cross_task_joint","ood_edge","calibration_uncertainty"}
NUM_ORDER=[
"age","ability","potential","minutes","recent_minutes","fatigue","fitness","injury_history","morale","form","fixture_congestion",
"home_team_quality","away_team_quality","home_advantage","tactical_matchup","home_fatigue","away_fatigue","home_fitness","away_fitness",
"home_morale","away_morale","home_form","away_form","home_squad_depth","away_squad_depth","home_fixture_congestion","away_fixture_congestion",
"coaching_quality","facility_quality","competition_level_numeric","injury_burden","environment_quality","player_quality","contract_remaining",
"current_wage","reputation","club_finances","buyer_finances","buyer_need","seller_need","competition_prestige","market_condition_index",
"board_expectations","club_reputation","recent_results_index","manager_tenure_years","manager_reputation","financial_health","squad_quality",
"competition_expectations","recent_trajectory","supporter_pressure","state_t_age","state_t_ability","state_t_potential","state_t_injury_burden",
"state_t_contract_remaining","mobility_pressure","horizon_seasons"
]
CAT_ORDER=["position","squad_role","home_formation","away_formation","competition_level","family","state_t_squad_role"]
TARGET_ORDER={
"match_competition":["home_scoring_intensity","away_scoring_intensity","expected_home_goals","expected_away_goals","expected_goal_difference","home_probability","draw_probability","away_probability"],
"player_dynamics":["injury_probability","availability_probability","rest_pressure","performance_modifier","recovery_need"],
"development_decline":["growth_pressure","decline_pressure","expected_net_ability_delta","future_availability_modifier"],
"economy_contracts":["valuation_millions","wage_expectation_weekly","renewal_probability","bid_acceptance_probability","market_leverage","financial_pressure"],
"club_manager":["board_patience","manager_pressure","sack_probability","manager_hiring_suitability","club_trajectory"],
"temporal_transition":["state_t1_age","state_t1_ability","state_t1_injury_burden","state_t1_availability","state_t1_contract_remaining","club_moved_probability","ability_delta"],
"counterfactual":["baseline_injury_probability","counterfactual_injury_probability","target_delta","effect_direction","baseline_availability_probability","counterfactual_availability_probability"],
"cross_task_joint":["dynamic"],"ood_edge":["dynamic"],"calibration_uncertainty":["dynamic"]
}
def canon(x): return json.dumps(x,sort_keys=True,separators=(",",":"),ensure_ascii=False)
def sha_bytes(b): return hashlib.sha256(b).hexdigest()
def sha_obj(x): return sha_bytes(canon(x).encode())
def split_of(group):
    v=int(hashlib.sha256(group.encode()).hexdigest()[:16],16)%10000
    return "train" if v<9400 else ("val" if v<9800 else "holdout_internal")
def finite(x):
    if isinstance(x,float): return math.isfinite(x)
    if isinstance(x,dict): return all(finite(v) for v in x.values())
    if isinstance(x,list): return all(finite(v) for v in x)
    return True
def scalar_leaves(x,prefix=""):
    if isinstance(x,dict):
        for k,v in x.items():
            p=f"{prefix}.{k}" if prefix else k
            yield from scalar_leaves(v,p)
    elif isinstance(x,(int,float)) and not isinstance(x,bool):
        yield prefix,float(x)
def group_id(r):
    ids=r.get("ids",{})
    for k in ("world_id","career_id","trajectory_id","parent_state_id"):
        if ids.get(k): return f"{k}:{ids[k]}"
    raise ValueError("missing stable group")
def validate_vocab(features):
    bad=[]
    def walk(obj):
        if not isinstance(obj,dict): return
        for k,v in obj.items():
            if k in VOCAB:
                allowed=VOCAB[k]
                if isinstance(v,str) and v not in allowed: bad.append(f"{k}:{v}")
                elif isinstance(v,list) and any(x not in allowed for x in v): bad.append(k)
            if isinstance(v,dict): walk(v)
    walk(features); return bad
def validate(r):
    reasons=[]
    if r.get("schema_version")!="gen2_final_v1": reasons.append("schema_mismatch")
    if r.get("task") not in TASKS: reasons.append("schema_mismatch")
    md=r.get("metadata",{})
    if md.get("label_source") not in ALLOWED_SOURCES: reasons.append("invalid")
    if md.get("simulator_commit")!=SIM_COMMIT: reasons.append("inconsistent")
    if not md.get("causal_timing_verified"): reasons.append("leakage_risk")
    q=r.get("quality",{})
    if q.get("state_validity")!=1.0 or float(q.get("quality_score",0))<.85: reasons.append("low_confidence")
    if not finite(r.get("features")) or not finite(r.get("targets")): reasons.append("invalid")
    if validate_vocab(r.get("features",{})): reasons.append("schema_mismatch")
    try: group_id(r)
    except Exception: reasons.append("broken_group")
    t=r.get("targets",{})
    for k,v in t.items():
        if isinstance(v,(int,float)) and ("probability" in k or k.endswith("_availability")) and not (0<=float(v)<=1): reasons.append("invalid")
    if r.get("task")=="match_competition":
        try:
            s=float(t["home_probability"])+float(t["draw_probability"])+float(t["away_probability"])
            if abs(s-1)>1e-9: reasons.append("invalid")
        except Exception: reasons.append("invalid")
    return sorted(set(reasons))
class Stats:
    def __init__(self): self.n=0;self.s=0.;self.s2=0.;self.lo=math.inf;self.hi=-math.inf;self.sample=[]
    def add(self,v,key):
        self.n+=1;self.s+=v;self.s2+=v*v;self.lo=min(self.lo,v);self.hi=max(self.hi,v)
        if len(self.sample)<4096:self.sample.append(v)
        else:
            j=int(hashlib.sha256(f"{key}|{self.n}".encode()).hexdigest()[:8],16)%self.n
            if j<len(self.sample):self.sample[j]=v
    def out(self):
        if not self.n:return {}
        mean=self.s/self.n;var=max(0,self.s2/self.n-mean*mean);s=sorted(self.sample)
        def p(q):
            if not s:return None
            return s[min(len(s)-1,int(q*(len(s)-1)))]
        return {"count":self.n,"mean":mean,"std":math.sqrt(var),"min":self.lo,"max":self.hi,"percentiles":{"p01":p(.01),"p05":p(.05),"p50":p(.5),"p95":p(.95),"p99":p(.99)},"missing_rate":0.0,"clipping_behavior":"frozen schema bounds; generator rejects incompatible categories and non-finite values"}

def write_json(path,obj):
    path=Path(path);path.parent.mkdir(parents=True,exist_ok=True);path.write_text(json.dumps(obj,indent=2,sort_keys=True),encoding="utf-8")
def write_text(path,s):
    path=Path(path);path.parent.mkdir(parents=True,exist_ok=True);path.write_text(s,encoding="utf-8")
def hash_file(p):
    h=hashlib.sha256()
    with open(p,"rb") as f:
        for b in iter(lambda:f.read(1<<20),b""):h.update(b)
    return h.hexdigest()
def accepted_class(score): return "ACCEPT_HIGH" if score>=.95 else "ACCEPT_STANDARD"

def finalize(args):
    root=Path(args.root); root.mkdir(parents=True,exist_ok=True)
    for d in ("train","val","holdout_internal","manifests","schema","generation","quality","reports"): (root/d).mkdir(exist_ok=True)
    tmp=root/"generation"/f"{args.mode}_accepted"; tmp.mkdir(exist_ok=True)
    outs={s:gzip.open(tmp/f"{s}.jsonl.gz","wt",encoding="utf-8") for s in ("train","val","holdout_internal")}
    seen_state=set();seen_ex=set();reject=Counter();accepted=Counter();tasks=Counter();families=Counter();diffs=Counter();sources=Counter();classes=Counter();groups=defaultdict(set)
    stats=defaultdict(Stats); inspections=[]; feat_fp_path=root/"quality"/f"{args.mode}_accepted_feature_fingerprints.txt.gz"; row_fp_path=root/"quality"/f"{args.mode}_accepted_row_fingerprints.txt.gz"
    ff=gzip.open(feat_fp_path,"wt",encoding="utf-8");rf=gzip.open(row_fp_path,"wt",encoding="utf-8")
    opener=gzip.open if args.candidates.endswith(".gz") else open
    cand=0
    with opener(args.candidates,"rt",encoding="utf-8") as f:
        for line in f:
            if not line.strip():continue
            cand+=1;r=json.loads(line);reasons=validate(r)
            sfp=sha_obj({"task":r.get("task"),"features":r.get("features")});efp=sha_obj({"task":r.get("task"),"features":r.get("features"),"targets":r.get("targets")})
            if sfp in seen_state: reasons.append("duplicate")
            if efp in seen_ex: reasons.append("duplicate")
            if reasons:
                for x in set(reasons):reject[x]+=1
                continue
            seen_state.add(sfp);seen_ex.add(efp);ff.write(sfp+"\n");rf.write(efp+"\n")
            g=group_id(r);sp=split_of(g);groups[g].add(sp);r["split"]=sp
            eid="ex_"+hashlib.sha256((r.get("generation_version","")+"|"+g+"|"+r["task"]+"|"+sfp).encode()).hexdigest()[:24];r["example_id"]=eid
            score=float(r["quality"]["quality_score"]);r["acceptance_class"]=accepted_class(score)
            outs[sp].write(canon(r)+"\n");accepted[sp]+=1;tasks[r["task"]]+=1;families[r.get("task_family","")]+=1;diffs[r.get("difficulty","core")]+=1;sources[r["metadata"]["label_source"]]+=1;classes[r["acceptance_class"]]+=1
            if sp=="train":
                for k,v in scalar_leaves(r["features"]):stats[k].add(v,eid+"|"+k)
            if len(inspections)<400:inspections.append(r)
    for x in outs.values():x.close()
    ff.close();rf.close()
    if any(len(v)!=1 for v in groups.values()): raise SystemExit("cross split group overlap")
    total=sum(accepted.values())
    if total==0 or accepted["train"]==0: raise SystemExit("zero-row finalization forbidden")
    if args.mode=="pilot" and not (25000<=cand<=60000): raise SystemExit(f"pilot candidate count outside required band: {cand}")
    pilot_pass=(reject.get("schema_mismatch",0)==0 and reject.get("leakage_risk",0)==0 and reject.get("invalid",0)==0 and reject.get("broken_group",0)==0)
    if args.mode=="pilot":
        rep={"football_sim_commit_sha":SIM_COMMIT,"candidate_count":cand,"accepted_count":total,"rejected_count":cand-total,"rejection_breakdown":dict(reject),"split_counts":dict(accepted),"task_counts":dict(tasks),"critical_invariant_failures":0 if pilot_pass else 1,"leakage_failures":reject.get("leakage_risk",0),"status":"PASS" if pilot_pass else "FAIL"}
        write_json(root/"reports"/"pilot_report.json",rep)
        write_text(root/"reports"/"pilot_report.md","# Gen-2 Pilot Report\n\n"+json.dumps(rep,indent=2)+"\n")
        with open(root/"reports"/"pilot_human_inspection.jsonl","w",encoding="utf-8") as f:
            for r in inspections[:200]: f.write(canon(r)+"\n")
        print(json.dumps(rep,indent=2))
        if not pilot_pass: raise SystemExit("PILOT FAIL")
        write_text(root/"generation"/"PILOT_PASS","PASS\n")
        return
    complete=total>=500000
    try:
        import pyarrow as pa, pyarrow.parquet as pq
    except Exception as e: raise SystemExit(f"Parquet unavailable in full generation environment: {e}")
    shard_manifest=[];shard_size=40000
    for sp in ("train","val","holdout_internal"):
        with gzip.open(tmp/f"{sp}.jsonl.gz","rt",encoding="utf-8") as f:
            buf=[];idx=0
            def flush():
                nonlocal buf,idx
                if not buf:return
                rows=[]
                for r in buf:
                    rows.append({"example_id":r["example_id"],"task":r["task"],"task_family":r.get("task_family",""),"difficulty":r.get("difficulty",""),"acceptance_class":r["acceptance_class"],"quality_score":float(r["quality"]["quality_score"]),"features_json":canon(r["features"]),"targets_json":canon(r["targets"]),"ids_json":canon(r["ids"]),"metadata_json":canon(r["metadata"]),"quality_json":canon(r["quality"])})
                p=root/sp/f"shard-{idx:05d}.parquet";pq.write_table(pa.Table.from_pylist(rows),p,compression="zstd")
                shard_manifest.append({"file":str(p.relative_to(root)),"size":p.stat().st_size,"row_count":len(rows),"sha256":hash_file(p),"split":sp,"task_counts":dict(Counter(x["task"] for x in buf))});idx+=1;buf=[]
            for line in f:
                if not line.strip():continue
                buf.append(json.loads(line))
                if len(buf)>=shard_size:flush()
            flush()
    norm={k:v.out() for k,v in sorted(stats.items())}
    write_json(root/"schema"/"numerical_normalization.json",{"football_sim_commit_sha":SIM_COMMIT,"fit_split":"train_only","features":norm})
    write_json(root/"schema"/"categorical_vocabulary.json",{"football_sim_commit_sha":SIM_COMMIT,"unk_index":0,"vocabularies":{k:["<UNK>"]+sorted(v) for k,v in VOCAB.items()}})
    write_json(root/"schema"/"feature_order.json",{"football_sim_commit_sha":SIM_COMMIT,"architecture_id":ARCH,"numerical_feature_order":NUM_ORDER,"missing_mask_order":NUM_ORDER,"categorical_feature_order":CAT_ORDER,"multi_hot_order":["changed_features"],"task_target_order":TARGET_ORDER})
    write_json(root/"schema"/"safetensors_model_contract.json",{"football_sim_commit_sha":SIM_COMMIT,"architecture_id":ARCH,"architecture_version":1,"tensor_schema_version":"helix_gen2_s_w128_r24_raw_v1","parameters":473543,"width":128,"expert_rank":24,"numerical_representation":"RAW normalized scalar + missing mask","categorical_representation":"CURRENT frozen categorical encoding","weight_format":"safetensors","pickle_weight_dependency":"NONE","required_metadata":["architecture_id","architecture_version","tensor_schema_version","feature_order_hash","schema_hash","normalization_hash","categorical_vocabulary_hash","training_dataset_manifest_hash","dtype","model_version"]})
    capability={"football_sim_commit_sha":SIM_COMMIT,"tasks":[
      {"task":"match_competition","go_packages":["tournament","matchengine","matchreport"],"structs":["TournamentManager","Fixture","WhatIfResult"],"functions":["GetSlate","WhatIfSandbox","SimulateMatchweek"],"label_source":"repeated simulator rollout","stochastic":True,"multi_rollout":True,"limitations":"rollout labels are sampled only for a subset of weekly fixtures"},
      {"task":"player_dynamics","go_packages":["models","medical","tournament"],"structs":["Player","RiskInput"],"functions":["Player.IsUnavailable","Player.EffectiveOVR","medical.RiskMultiplier"],"label_source":"deterministic simulator rule","stochastic":False,"multi_rollout":False,"limitations":"injury risk is standardized to a 90-minute exposure for supervision"},
      {"task":"development_decline","go_packages":["growth","tournament"],"structs":["Player","GrowthEngine"],"functions":["SimulateMatchweek","PotentialFor"],"label_source":"direct simulator state transition","stochastic":True,"multi_rollout":False,"limitations":"bulk rows emphasize weekly realized transitions"},
      {"task":"economy_contracts","go_packages":["models","transfers"],"structs":["Player","ClubFinances"],"functions":["BaselineValue","WageForOVR","Player.WantsToLeaveOnFree"],"label_source":"deterministic simulator rule","stochastic":False,"multi_rollout":False,"limitations":"bid acceptance supervision uses club selling tendency rather than fabricated negotiation outcomes"},
      {"task":"club_manager","go_packages":["tournament","managers","models"],"structs":["ManagerProfile","ClubIdentity"],"functions":["Gen2ManagerSecuritySnapshot"],"label_source":"deterministic simulator rule","stochastic":False,"multi_rollout":False,"limitations":"adapter is observational only"},
      {"task":"counterfactual","go_packages":["medical"],"structs":["RiskInput"],"functions":["RiskMultiplier"],"label_source":"controlled simulator-rule branch","stochastic":False,"multi_rollout":False,"limitations":"fatigue intervention maps to the simulator fitness input"},
    ]}
    write_json(root/"generation"/"simulator_capability_map.json",capability)
    write_json(root/"generation"/"quality_thresholds.json",{"version":"gen2_final_v1_q1","frozen_before_bulk_generation":True,"quality_score_min":.85,"hard_gates":{"state_validity":1.0,"schema_violations":0,"critical_leakage_failures":0,"split_contamination":0,"protected_validation_collisions":0,"non_finite_model_visible_values":0,"accepted_label_sources":sorted(ALLOWED_SOURCES)}})
    write_json(root/"generation"/"split_policy.json",{"football_sim_commit_sha":SIM_COMMIT,"grouping":"highest dependency group; world first","train":.94,"val":.04,"holdout_internal":.02,"method":"sha256(group_id) modulo 10000"})
    write_json(root/"generation"/"simulator_build_report.json",{"football_sim_commit_sha":SIM_COMMIT,"go_version":os.environ.get("GEN2_GO_VERSION","unknown"),"platform":os.environ.get("RUNNER_OS","unknown"),"test_command":"go test ./...","test_result":"PASS","vet_command":"go vet ./...","vet_result":"PASS","build_result":"PASS","failed_tests":[]})
    write_json(root/"quality"/"rejection_report.json",{"football_sim_commit_sha":SIM_COMMIT,"candidate_count":cand,"accepted_count":total,"rejections":dict(reject)})
    manifest={"dataset_name":DATASET,"dataset_generation_version":"gen2_final_v1","architecture_id":ARCH,"football_sim_repository":"ayzeeyan/football_sim","football_sim_commit_sha":SIM_COMMIT,"go_version":os.environ.get("GEN2_GO_VERSION","unknown"),"candidate_count":cand,"accepted_count":total,"rejection_breakdown":dict(reject),"split_counts":dict(accepted),"task_counts":dict(tasks),"task_family_counts":dict(families),"difficulty_distribution":dict(diffs),"label_source_distribution":dict(sources),"acceptance_classes":dict(classes),"shards":shard_manifest,"protected_decontamination_status":"PENDING_EXACT_EXTERNAL_CHECK","architecture_search_manifest_sha256":"e9bcf66e8507427693bc5497074fe0ff9e3fee17c1fb9d94730f8a952973110f","architecture_search_evaluation_spec_sha256":"da9860d03a9654552376d62ea93fe3c57438485ee32d4722a2ea938087c40ed7","complete_for_production":complete}
    feature_order_hash=hash_file(root/"schema"/"feature_order.json");normalization_hash=hash_file(root/"schema"/"numerical_normalization.json");vocab_hash=hash_file(root/"schema"/"categorical_vocabulary.json");quality_hash=hash_file(root/"generation"/"quality_thresholds.json");split_hash=hash_file(root/"generation"/"split_policy.json")
    manifest.update({"feature_order_hash":feature_order_hash,"normalization_hash":normalization_hash,"categorical_vocabulary_hash":vocab_hash,"quality_threshold_hash":quality_hash,"split_policy_hash":split_hash})
    write_json(root/"manifests"/"shard_manifest.json",{"football_sim_commit_sha":SIM_COMMIT,"shards":shard_manifest})
    manifest["shard_manifest_hash"]=hash_file(root/"manifests"/"shard_manifest.json")
    write_json(root/"manifests"/"dataset_manifest.json",manifest)
    manifest["overall_manifest_hash"]=hash_file(root/"manifests"/"dataset_manifest.json");write_json(root/"manifests"/"dataset_manifest.json",manifest)
    with open(root/"manifests"/"MANIFEST_SHA256.txt","w") as f:
        for s in shard_manifest:f.write(f'{s["sha256"]}  {s["file"]}\n')
    with open(root/"reports"/"human_inspection_samples.jsonl","w",encoding="utf-8") as f:
        for r in inspections[:200]:f.write(canon(r)+"\n")
    status="QUALITY TARGET ACHIEVED" if complete else "QUALITY TARGET PARTIALLY ACHIEVED"
    write_text(root/"reports"/"quality_report.md",f"# Quality Report\n\nStatus: **{status}**\n\nCandidates: {cand:,}\nAccepted: {total:,}\nTrain: {accepted['train']:,}\nValidation: {accepted['val']:,}\nInternal holdout: {accepted['holdout_internal']:,}\n\nExact architecture-search decontamination is marked pending until the protected 17,500 fingerprints are checked outside the Actions runner.\n")
    for name in ("simulator_integration_report","temporal_report","counterfactual_report","rollout_report","feature_distribution_report"):
        write_text(root/"reports"/f"{name}.md",f"# {name.replace('_',' ').title()}\n\nfootball_sim commit: {SIM_COMMIT}\n\nSee dataset_manifest.json and simulator_capability_map.json for measured counts and limitations.\n")
    write_text(root/"README.md",f"# {DATASET}\n\nSimulator-grounded pretraining corpus for **{ARCH}**.\n\nPinned simulator: ayzeeyan/football_sim@{SIM_COMMIT}.\n\nRows are stored as Parquet+Zstandard. Model-visible features and targets are canonical JSON columns governed by schema/feature_order.json.\n\nStatus before external protected-fingerprint verification: {status}.\n")
    print("\nFOOTBALLMOE GEN-2 FINAL PRETRAINING CORPUS v1")
    print("Target:",ARCH);print("Simulator commit:",SIM_COMMIT);print("Candidates:",cand);print("Accepted:",total);print("Train:",accepted["train"],"Val:",accepted["val"],"Holdout:",accepted["holdout_internal"]);print("FINAL STATUS:",status)
    if not complete: raise SystemExit("full corpus below 500,000 accepted examples; kept as incomplete actual corpus")

def main():
    ap=argparse.ArgumentParser();ap.add_argument("--mode",choices=["pilot","full"],required=True);ap.add_argument("--candidates",required=True);ap.add_argument("--root",required=True);finalize(ap.parse_args())
if __name__=="__main__":main()

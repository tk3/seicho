(function(root){
function gitBadge(status){return status?.repository&&status.changes.length?' ● '+status.changes.length:''}
function preferredDiffStaged(change){return !change.unstaged&&change.staged}
function hasStagedChanges(status){return Boolean(status?.changes?.some(change=>change.staged))}
function nextGitPanelOpenState(panelIsHidden){return panelIsHidden}
function createGitPanel(options){
 const elements={panel:document.querySelector('#git-panel'),toggle:document.querySelector('#git-toggle'),count:document.querySelector('#git-count'),close:document.querySelector('#git-close'),state:document.querySelector('#git-repository-state'),branch:document.querySelector('#git-branch'),content:document.querySelector('#git-content'),changes:document.querySelector('#git-changes'),diff:document.querySelector('#git-diff'),error:document.querySelector('#git-error'),feedback:document.querySelector('#git-feedback'),message:document.querySelector('#git-commit-message'),commit:document.querySelector('#git-commit'),commitHint:document.querySelector('#git-commit-hint')};
 let status={repository:false,branch:'',changes:[]};
 let feedbackTimer=0;
 const t=key=>options.translate(options.getLocale(),key);
 function setError(message=''){elements.error.value=message}
 function setFeedback(message='',autoHide=false){clearTimeout(feedbackTimer);feedbackTimer=0;elements.feedback.value=message;if(message&&autoHide)feedbackTimer=setTimeout(()=>{elements.feedback.value='';feedbackTimer=0},3000)}
 function setOpen(open){elements.panel.classList.toggle('hidden',!open);elements.toggle.setAttribute('aria-expanded',String(open));if(open)refresh().catch(()=>{})}
 function render(){
  elements.count.textContent=gitBadge(status);
  elements.content.classList.toggle('hidden',!status.repository);
  elements.state.textContent=status.repository?'':t('gitNotRepository');
  elements.branch.textContent=status.repository?t('gitBranch')+': '+(status.branch||'-'):'';
  elements.branch.classList.toggle('hidden',!status.repository);
  elements.commit.disabled=!hasStagedChanges(status);
  elements.commit.title=elements.commit.disabled?t('gitNothingStaged'):'';
  elements.commitHint.classList.toggle('hidden',!elements.commit.disabled);
  elements.changes.innerHTML='';
  if(!status.repository)return;
  if(!status.changes.length){const empty=document.createElement('div');empty.className='git-empty';empty.textContent=t('gitNoChanges');elements.changes.append(empty);return}
  status.changes.forEach(change=>{
   const row=document.createElement('div');row.className='git-change';
   const main=document.createElement('button');main.type='button';main.className='git-change-main';main.title=change.path;
   const code=document.createElement('span');code.className='git-change-status';code.textContent=change.status;
   const path=document.createElement('span');path.className='git-change-path';path.textContent=change.path;
   main.append(code,path);main.onclick=()=>loadDiff(change,preferredDiffStaged(change));
   const actions=document.createElement('div');actions.className='git-change-actions';
   if(change.unstaged)actions.append(actionButton(t('gitStage'),()=>stage(change.path,true)));
   if(change.staged)actions.append(actionButton(t('gitUnstage'),()=>stage(change.path,false)));
   row.append(main,actions);elements.changes.append(row);
  });
 }
 function actionButton(label,action){const button=document.createElement('button');button.type='button';button.textContent=label;button.onclick=action;return button}
 async function refresh(){
  try{setError();status=await options.api('/api/git/status');render();return status}
  catch(error){elements.count.textContent=' !';setError(error.message);throw error}
 }
 async function loadDiff(change,staged){
  try{setError();const result=await options.api('/api/git/diff?path='+encodeURIComponent(change.path)+'&staged='+staged);elements.diff.textContent=result.diff||t('gitNoDiff')}
  catch(error){setError(error.message)}
 }
 async function stage(path,stageFile){
  if(options.hasUnsavedChanges()){setError(t('gitSaveFirst'));return}
  try{setError();setFeedback();await options.api('/api/git/stage',{method:'POST',body:JSON.stringify({path,stage:stageFile})});elements.diff.textContent='';await refresh()}
  catch(error){setError(error.message)}
 }
 async function commit(){
  if(options.hasUnsavedChanges()){setError(t('gitSaveFirst'));return}
  const message=elements.message.value.trim();
  try{setError();setFeedback();await options.api('/api/git/commit',{method:'POST',body:JSON.stringify({message})});elements.message.value='';elements.diff.textContent='';setFeedback(t('gitCommitted'),true);await refresh()}
  catch(error){setError(error.message)}
 }
 elements.toggle.onclick=()=>setOpen(nextGitPanelOpenState(elements.panel.classList.contains('hidden')));elements.close.onclick=()=>setOpen(false);elements.commit.onclick=commit;
 document.addEventListener('keydown',event=>{if(event.key==='Escape'&&!elements.panel.classList.contains('hidden'))setOpen(false)});
 render();
 return {refresh,render,open:()=>setOpen(true),close:()=>setOpen(false)};
}
const api={gitBadge,preferredDiffStaged,hasStagedChanges,nextGitPanelOpenState,createGitPanel};
if(typeof module==='object'&&module.exports)module.exports=api;
else Object.assign(root,api);
})(typeof globalThis==='object'?globalThis:this);

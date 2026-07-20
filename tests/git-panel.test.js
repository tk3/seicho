const test=require('node:test');
const assert=require('node:assert/strict');
const {gitBadge,preferredDiffStaged,hasStagedChanges,nextGitPanelOpenState,closeToolsMenu}=require('../web/git-panel.js');

test('shows a Git badge only when a repository has changes',()=>{
 assert.equal(gitBadge({repository:true,changes:[{},{}]}),' ● 2');
 assert.equal(gitBadge({repository:true,changes:[]}), '');
 assert.equal(gitBadge({repository:false,changes:[{}]}), '');
});

test('prefers the staged diff only when no unstaged change exists',()=>{
 assert.equal(preferredDiffStaged({staged:true,unstaged:false}),true);
 assert.equal(preferredDiffStaged({staged:true,unstaged:true}),false);
 assert.equal(preferredDiffStaged({staged:false,unstaged:true}),false);
});

test('enables commits only when at least one change is staged',()=>{
 assert.equal(hasStagedChanges({changes:[{staged:false}]}),false);
 assert.equal(hasStagedChanges({changes:[{staged:false},{staged:true}]}),true);
 assert.equal(hasStagedChanges({changes:[]}),false);
});

test('toggles the Git panel from its current visibility',()=>{
 assert.equal(nextGitPanelOpenState(true),true);
 assert.equal(nextGitPanelOpenState(false),false);
});

test('closes the tools menu after choosing Git',()=>{
 const menu={open:true};
 closeToolsMenu(menu);
 assert.equal(menu.open,false);
});

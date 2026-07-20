(function(root){
const messages={
ja:{language:'言語',closeDialog:'ダイアログを閉じる',close:'閉じる',openHugoSite:'Hugoサイトを開く',siteHelp:'サイト設定ファイルがあるフォルダーの絶対パスを入力してください。',siteFolder:'サイトフォルダー',openSite:'サイトを開く',newPost:'新しい投稿',newPostHelp:'contentフォルダーからの相対パスを入力してください。Hugoのarchetypeを使って作成します。',postPath:'投稿ファイルのパス',cancel:'キャンセル',createPost:'投稿を作成',deleteQuestion:'この投稿を削除しますか？',deleteHelp:'この操作は取り消せません。次のファイルをHugoサイトから削除します。',deletePost:'投稿を削除',unsavedChanges:'保存していない変更があります',unsavedHelp:'このまま移動すると、現在の編集内容は保存されません。',leaveWithoutSaving:'保存せずに移動',returnToEditing:'編集に戻る',siteNotSelected:'サイト未選択',changeSite:'サイト変更',sortPosts:'投稿を並び替え',sortOrder:'投稿の並び順',modifiedNewest:'更新日時が新しい順',modifiedOldest:'更新日時が古い順',dateNewest:'公開日が新しい順',searchPosts:'投稿を検索…',chooseWriting:'文章を書く場所を選びましょう',chooseWritingHelp:'左の一覧から投稿を選ぶか、新しい投稿を作成してください。',deleteThisPost:'この投稿を削除',delete:'削除',save:'保存',zenMode:'Zen Mode',exitZenMode:'Zen Modeを終了',filePath:'ファイルパス',startWriting:'ここから本文を書き始めます…',updated:'更新',created:'Hugoのarchetypeから投稿を作成しました',previewEmpty:'プレビューがここに表示されます。',previewError:'プレビューエラー',saved:'保存しました',deleted:'削除しました',postNotFound:'URLで指定された投稿が見つかりません',git:'Git',gitPanel:'Git',closeGit:'Gitパネルを閉じる',gitBranch:'ブランチ',gitChanges:'変更',gitDiff:'差分',gitNoChanges:'変更はありません',gitNotRepository:'選択したサイトはGitリポジトリではありません',gitStage:'ステージ',gitUnstage:'解除',gitNoDiff:'表示できる差分はありません',gitCommitMessage:'コミットメッセージ',gitCommit:'コミット',gitCommitted:'コミットしました',gitSaveFirst:'先に編集中の内容を保存してください'},
en:{language:'Language',closeDialog:'Close dialog',close:'Close',openHugoSite:'Open a Hugo site',siteHelp:'Enter the absolute path to the folder containing your Hugo site configuration.',siteFolder:'Site folder',openSite:'Open site',newPost:'New Post',newPostHelp:'Enter a path relative to the content folder. The post will be created from a Hugo archetype.',postPath:'Post file path',cancel:'Cancel',createPost:'New Post',deleteQuestion:'Delete this post?',deleteHelp:'This action cannot be undone. The following file will be deleted from the Hugo site.',deletePost:'Delete post',unsavedChanges:'You have unsaved changes',unsavedHelp:'Your current edits will not be saved if you continue.',leaveWithoutSaving:'Leave without saving',returnToEditing:'Return to editing',siteNotSelected:'No site selected',changeSite:'Change site',sortPosts:'Sort posts',sortOrder:'Post sort order',modifiedNewest:'Recently modified',modifiedOldest:'Least recently modified',dateNewest:'Newest publication date',searchPosts:'Search posts…',chooseWriting:'Choose something to write',chooseWritingHelp:'Select a post from the list or create a new one.',deleteThisPost:'Delete this post',delete:'Delete',save:'Save',zenMode:'Zen Mode',exitZenMode:'Exit Zen Mode',filePath:'File path',startWriting:'Start writing here…',updated:'Updated',created:'Created the post from a Hugo archetype',previewEmpty:'The preview will appear here.',previewError:'Preview error',saved:'Saved',deleted:'Deleted',postNotFound:'The post specified by the URL could not be found.',git:'Git',gitPanel:'Git',closeGit:'Close Git panel',gitBranch:'Branch',gitChanges:'Changes',gitDiff:'Diff',gitNoChanges:'No changes',gitNotRepository:'The selected site is not a Git repository',gitStage:'Stage',gitUnstage:'Unstage',gitNoDiff:'No diff is available',gitCommitMessage:'Commit message',gitCommit:'Commit',gitCommitted:'Committed',gitSaveFirst:'Save the current edits first'}
};
messages.ja.gitNothingStaged='コミットする変更をステージしてください';
messages.en.gitNothingStaged='Stage at least one change before committing';
messages.ja.gitCommitHint='変更をステージするとコミットできます';
messages.en.gitCommitHint='Stage a change to enable commits';
messages.ja.tools='ツール';
messages.en.tools='Tools';
messages.ja.postActions='記事の操作';
messages.en.postActions='Post actions';
messages.ja.switchToLight='ライトテーマに切り替える';
messages.en.switchToLight='Switch to light theme';
messages.ja.switchToDark='ダークテーマに切り替える';
messages.en.switchToDark='Switch to dark theme';
function translate(locale,key){return messages[locale]?.[key]||key}
const api={messages,translate};
if(typeof module==='object'&&module.exports)module.exports=api;
else{root.seichoMessages=messages;root.translate=translate}
})(typeof globalThis==='object'?globalThis:this);

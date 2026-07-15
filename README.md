# Seicho

Hugo の `content` フォルダーをブラウザから編集する、ローカル専用の軽量エディターです。

## 起動

```bash
go run .
```

ブラウザで <http://127.0.0.1:1314> を開き、Hugoサイトの絶対パスを入力します。起動時に指定することもできます。

```bash
go run . -site /c/path/to/hugo-site
```

実行ファイルを作る場合は次を実行します。

```bash
go build -buildvcs=false -o seicho.exe .
```

ビルド済みの実行ファイルはGit Bashから次のように起動できます。

```bash
./seicho.exe
```

Hugoサイトを指定して起動する場合：

```bash
./seicho.exe -site /c/path/to/hugo-site
```

## 機能

- Markdown投稿の一覧・検索・更新日時順の並べ替え
- `hugo new content`による新規投稿作成（`archetypes/default.md`対応）
- YAML/TOML Front Matterを保った編集
- 新規投稿、保存、削除
- Markdownの簡易ライブプレビュー
- 外部変更との上書き競合検出
- `content` 外へのファイル操作を防止

現時点ではHugoプレビューサーバーの起動、画像アップロード、Git操作は未対応です。

新規投稿の作成では、`hugo`コマンドへPATHが通っている必要があります。選択したサイトを作業ディレクトリとして、次の形式のコマンドを実行します。

```bash
hugo new content posts/example.md
```

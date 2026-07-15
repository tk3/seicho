# Seicho

Hugo の `content` フォルダーをブラウザから編集する、ローカル専用の軽量エディターです。

## 起動

```bash
go run . -port 1314
```

ブラウザで <http://127.0.0.1:1314> を開き、Hugoサイトの絶対パスを入力します。起動時に指定することもできます。

```bash
go run . -site /path/to/hugo-site
```

実行ファイルを作る場合は次を実行します。

```bash
go build -buildvcs=false -o seicho .
```

ビルド済みの実行ファイルはGit Bashから次のように起動できます。

```bash
./seicho -port 1314
```

Hugoサイトを指定して起動する場合：

```bash
./seicho -site /path/to/hugo-site
```

ポート番号を変更する場合は`-port`を指定します。未指定時は`1314`です。

```bash
./seicho -port 8080
```

サイトとポート番号は同時に指定できます。

```bash
./seicho -site /path/to/hugo-site -port 8080
```

バージョン情報を確認する場合：

```bash
./seicho -version
```

## 機能

- Markdown投稿の一覧・検索・更新日時順の並べ替え
- `hugo new content`による新規投稿作成（`archetypes/default.md`対応）
- YAML/TOML Front Matterを保った編集
- 新規投稿、保存、削除
- Hugoと同じGoldmarkを使ったMarkdownライブプレビュー
- 外部変更との上書き競合検出
- `content` 外へのファイル操作を防止

現時点ではHugoプレビューサーバーの起動、画像アップロード、Git操作は未対応です。

簡易プレビューはHugoと同じGoldmarkパーサーを使用します。CommonMark、テーブル、取り消し線、タスクリスト、定義リスト、脚注などに対応します。Shortcode、テーマ、Render Hookを含む最終表示の再現には、将来対応予定のHugoプレビューサーバーが必要です。

新規投稿の作成では、`hugo`コマンドへPATHが通っている必要があります。選択したサイトを作業ディレクトリとして、次の形式のコマンドを実行します。

```bash
hugo new content posts/example.md
```

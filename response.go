package main

import (
	"encoding/json"
	"errors"
	"net/http"
)

func decodeJSON(r *http.Request, value any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 4<<20)
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(value); err != nil {
		return codedError("invalid_request", err)
	}
	return nil
}

func jsonResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func apiError(w http.ResponseWriter, status int, err error) {
	if recorder, ok := w.(interface{ recordError(error) }); ok {
		recorder.recordError(err)
	}
	code := "internal_error"
	var failure *apiFailure
	if errors.As(err, &failure) {
		code = failure.code
	}
	language := "ja"
	if localized, ok := w.(interface{ responseLanguage() string }); ok {
		language = localized.responseLanguage()
	}
	response := map[string]string{"code": code, "error": errorMessage(code, language)}
	if traced, ok := w.(interface{ traceRequestID() string }); ok {
		response["requestId"] = traced.traceRequestID()
	}
	jsonResponse(w, status, response)
}

type apiFailure struct {
	code  string
	cause error
}

func (e *apiFailure) Error() string {
	if e.cause == nil {
		return e.code
	}
	return e.code + ": " + e.cause.Error()
}

func (e *apiFailure) Unwrap() error { return e.cause }

func codedError(code string, cause error) error {
	return &apiFailure{code: code, cause: cause}
}

type localizedMessages struct {
	ja string
	en string
}

var apiErrorMessages = map[string]localizedMessages{
	"internal_error":                  {"サーバー内部で予期しないエラーが発生しました。", "An unexpected internal server error occurred."},
	"invalid_request":                 {"リクエストの内容が正しくありません。", "The request is invalid."},
	"preview_failed":                  {"Markdownをプレビューできません。", "Could not preview the Markdown."},
	"invalid_site_path":               {"サイトフォルダーのパスが正しくありません。", "The site folder path is invalid."},
	"site_folder_not_found":           {"指定したフォルダーが見つかりません。", "The specified folder was not found."},
	"hugo_config_not_found":           {"Hugoサイト設定ファイルが見つかりません。", "No Hugo site configuration file was found."},
	"create_content_directory_failed": {"contentフォルダーを作成できません。", "Could not create the content folder."},
	"site_not_selected":               {"Hugoサイトを選択してください。", "Select a Hugo site first."},
	"list_posts_failed":               {"投稿の一覧を読み込めません。", "Could not load the post list."},
	"read_post_failed":                {"投稿ファイルを読み込めません。", "Could not read the post file."},
	"read_file_info_failed":           {"投稿ファイルの情報を取得できません。", "Could not read the post file information."},
	"post_already_exists":             {"同じパスの投稿がすでに存在します。", "A post already exists at the same path."},
	"hugo_new_timeout":                {"hugo newがタイムアウトしました。", "hugo new timed out."},
	"hugo_new_failed":                 {"hugo newの実行に失敗しました。", "Failed to run hugo new."},
	"read_generated_post_failed":      {"hugo newは成功しましたが、生成ファイルを読み込めません。", "hugo new succeeded, but the generated file could not be read."},
	"file_modified":                   {"保存後にファイルが変更されています。再読み込みしてください。", "The file has been modified since it was loaded. Reload it and try again."},
	"destination_exists":              {"変更先のパスには既に投稿が存在します。", "A post already exists at the destination path."},
	"inspect_destination_failed":      {"変更先のファイルを確認できません。", "Could not inspect the destination file."},
	"create_post_directory_failed":    {"投稿先のフォルダーを作成できません。", "Could not create the post folder."},
	"write_post_failed":               {"投稿ファイルを書き込めません。", "Could not write the post file."},
	"replace_post_failed":             {"投稿ファイルを保存できません。", "Could not save the post file."},
	"verify_saved_post_failed":        {"保存後のファイルを確認できません。", "Could not verify the saved file."},
	"saved_content_mismatch":          {"保存後のファイル内容が一致しません。", "The saved file content does not match the requested content."},
	"remove_original_post_failed":     {"元ファイルを削除できないため名前を変更できません。", "Could not rename the post because the original file could not be removed."},
	"delete_post_failed":              {"投稿ファイルを削除できません。", "Could not delete the post file."},
	"invalid_post_path":               {"投稿パスが不正です。", "The post path is invalid."},
	"invalid_post_extension":          {"拡張子は .md または .markdown にしてください。", "Use the .md or .markdown file extension."},
	"path_outside_content":            {"contentフォルダー外は操作できません。", "Files outside the content folder cannot be accessed."},
	"git_not_installed":               {"gitコマンドが見つかりません。", "The git command could not be found."},
	"git_repository_not_found":        {"選択したHugoサイトはGitリポジトリではありません。", "The selected Hugo site is not a Git repository."},
	"git_status_failed":               {"Gitの状態を取得できません。", "Could not read the Git status."},
	"git_diff_failed":                 {"Gitの差分を取得できません。", "Could not read the Git diff."},
	"git_stage_failed":                {"ファイルをステージできません。", "Could not stage the file."},
	"git_unstage_failed":              {"ファイルのステージを解除できません。", "Could not unstage the file."},
	"git_commit_message_required":     {"コミットメッセージを入力してください。", "Enter a commit message."},
	"git_nothing_staged":              {"コミットする変更をステージしてください。", "Stage at least one change before committing."},
	"git_commit_failed":               {"コミットできません。", "Could not create the commit."},
	"git_invalid_path":                {"Gitで操作するファイルパスが不正です。", "The file path for the Git operation is invalid."},
	"method_not_allowed":              {"許可されていない操作です。", "This operation is not allowed."},
}

func errorMessage(code, language string) string {
	messages, ok := apiErrorMessages[code]
	if !ok {
		messages = apiErrorMessages["internal_error"]
	}
	if language == "en" {
		return messages.en
	}
	return messages.ja
}

func methodNotAllowed(w http.ResponseWriter) {
	apiError(w, http.StatusMethodNotAllowed, codedError("method_not_allowed", nil))
}

package main

import (
	"testing"
)

func TestAvatarURL(t *testing.T) {
	var authAvatar AuthAvatar
	client := new(client)
	url, err := authAvatar.GetAvatarURL(client)
	if err != ErrNoAvatarURL {
		t.Error("値が存在しない場合、AuthAvatar.GetAvatarURL は ErrNoAvatarURL を返すべきです。")
	}
	testURL := "http://url-to-avatar/"
	client.userData = map[string]interface{}{
		"avatar_url": testURL,
	}
	url, err = authAvatar.GetAvatarURL(client)
	if err != nil {
		t.Error("値が存在する場合、AuthAvatar.GetAvatarURL はエラーを返すべきではありません。")
	} else {
		if url != testURL {
			t.Errorf("取得したURLが正しくありません。期待値: %s、取得値: %s。", testURL, url)
		}
	}
}

func TestGravatarAvatar(t *testing.T){
	var gravatarAvatar GravatarAvatar
	client := new(client)
	client.userData = map[string]interface{}{
		"email": "MyEmailAddress@example.com",
	}
	url, err := gravatarAvatar.GetAvatarURL(client)
	if err != nil {
		t.Error("GravatarAvatar.GetAvatarURLはエラーを返すべきではありません")
	}
	if url != "//www.gravatar.com/avatar/0bc83cb571cd1c50ba6f3e8a78ef1346" {
		t.Errorf("GravatarAvatar.GetAvatarURLが正しいURLを返しませんでした。期待値: %s、取得値: %s", "//www.gravatar.com/avatar/0bc83cb571cd1c50ba6f3e8a78ef1346", url)
	}
}
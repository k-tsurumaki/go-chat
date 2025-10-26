package main

import (
	"errors"
)

var ErrNoAvatarURL = errors.New("chat: アバターのURLを取得できません。")

type Avatar interface {
	GetAvatarURL(c *client) (string, error)
}

type AuthAvatar struct{}

var UseAuthAvatar AuthAvatar

func (AuthAvatar) GetAvatarURL(c *client) (string, error) {
	// AuthAvatar は OAuth プロバイダ等が返す avatar_url を利用してアバター URL を返す
	// - クライアントの userData に `avatar_url` があればそれを返す
	// - なければ ErrNoAvatarURL を返す
	if url, ok := c.userData["avatar_url"]; ok {
		if urlStr, ok := url.(string); ok {
			return urlStr, nil
		}
	}
	return "", ErrNoAvatarURL
}

type GravatarAvatar struct{}

var UseGravatar GravatarAvatar

func (GravatarAvatar) GetAvatarURL(c *client) (string, error) {
	// GravatarAvatar はユーザのメールアドレス（userData["email"]）を MD5 ハッシュ化して
	// Gravatar の URL を生成して返す
	// - userData に email が文字列で含まれていれば利用する
	// - なければ ErrNoAvatarURL を返す
	if userid, ok := c.userData["userid"]; ok {
		if useridStr, ok := userid.(string); ok {
			return "//www.gravatar.com/avatar/" + useridStr, nil
		}
	}
	return "", ErrNoAvatarURL
}

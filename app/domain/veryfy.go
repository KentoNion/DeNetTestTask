package domain

import "strings"

func VerifyEmail(email Email) error { //todo Я хотел ещё проверитьчтобы email заканчивался на существующий домен верхнего типа, но потом загуглил и узнал что их больше 1500, хотел просто сделать мапу с ключами в виде названий всех доменов и значениями true bool и проверять что email заканчивается на стрингу которая есть в map, но их 1500 так что пускай пока без них, просто проверяем наличие собаки
	contains := strings.Contains(string(email), "@")
	if !contains {
		return ErrNotEmail
	}
	return nil
}

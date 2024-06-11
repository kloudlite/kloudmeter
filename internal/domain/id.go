package domain

func MeterKey(id string) string {
	return "meter-" + id
}

func EventKey(id string) string {
	return "event-" + id
}

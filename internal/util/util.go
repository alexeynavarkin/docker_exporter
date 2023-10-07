package util

const (
	LabelNameServiceName = "com.docker.swarm.service.name"
	LabelNameServiceID   = "com.docker.swarm.service.id"
	LabelDefaultValue    = "unknown"
)

func GetMapValue(_map map[string]string, key string, defaultValue string) string {
	val := _map[key]
	if val == "" {
		return defaultValue
	}
	return val
}

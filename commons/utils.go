package commons

import "github.com/posteo/go-agentx/value"

// Taken from "github.com/gin-gonic/gin"
func filterFlags(content string) string {
	for i, char := range content {
		if char == ' ' || char == ';' {
			return content[:i]
		}
	}
	return content
}

// Wrapper de value.ParseOID que no acepta que
// el oid arranque con "."
func ParseOID(oid string) (value.OID, error) {
	if len(oid) > 0 && oid[0] == '.' {
		return value.ParseOID(oid[1:])
	}
	return value.ParseOID(oid)
}

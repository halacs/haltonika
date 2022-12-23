package fmb920

import "strings"

func (s *Server) isAllowedIMEI(imei string) bool {
	for _, actualIMEI := range s.allowedIMEIs {
		if strings.EqualFold(imei, actualIMEI) {
			return true
		}
	}

	return false
}

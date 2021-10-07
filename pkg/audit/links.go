package audit

import "github.com/google/uuid"

type AssumeRoleLink struct {
	ID        string `json:"id"`
	SourceARN string `json:"sourceARN"`
	TargetARN string `json:"targetARN"`
}

// BuildLinks looks through all roles in the cache and checks whether
// the source role can assume the target role.
//
// Note: initial implementation is O(N^2) efficiency
func (a *Auditor) BuildLinks() {
	links := []AssumeRoleLink{}

	roles := a.roleStorage.List()

	for _, source := range roles {
		for _, target := range roles {
			// don't compare the source against itself
			if source.ARN == target.ARN {
				continue
			}
			if source.CanAssume(target) {
				link := AssumeRoleLink{
					ID:        uuid.NewString(),
					SourceARN: source.ARN,
					TargetARN: target.ARN,
				}
				links = append(links, link)
			}
		}
	}
	a.links = links
}

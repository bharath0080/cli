package v3action

import (
	"code.cloudfoundry.org/cli/actor/actionerror"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3/constant"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccversion"
	"github.com/blang/semver"
)

type Space struct {
	GUID             string
	Name             string
	OrganizationGUID string
}

// ResetSpaceIsolationSegment disassociates a space from an isolation segment.
//
// If the space's organization has a default isolation segment, return its
// name. Otherwise return the empty string.
func (actor Actor) ResetSpaceIsolationSegment(orgGUID string, spaceGUID string) (string, Warnings, error) {
	var allWarnings Warnings

	_, apiWarnings, err := actor.CloudControllerClient.UpdateSpaceIsolationSegmentRelationship(spaceGUID, "")
	allWarnings = append(allWarnings, apiWarnings...)
	if err != nil {
		return "", allWarnings, err
	}

	isoSegRelationship, apiWarnings, err := actor.CloudControllerClient.GetOrganizationDefaultIsolationSegment(orgGUID)
	allWarnings = append(allWarnings, apiWarnings...)
	if err != nil {
		return "", allWarnings, err
	}

	var isoSegName string
	if isoSegRelationship.GUID != "" {
		isolationSegment, apiWarnings, err := actor.CloudControllerClient.GetIsolationSegment(isoSegRelationship.GUID)
		allWarnings = append(allWarnings, apiWarnings...)
		if err != nil {
			return "", allWarnings, err
		}
		isoSegName = isolationSegment.Name
	}

	return isoSegName, allWarnings, nil
}

func (actor Actor) GetSpaceByNameAndOrganization(spaceName string, orgGUID string) (Space, Warnings, error) {
	spaces, warnings, err := actor.CloudControllerClient.GetSpaces(
		ccv3.Query{Key: ccv3.NameFilter, Values: []string{spaceName}},
		ccv3.Query{Key: ccv3.OrganizationGUIDFilter, Values: []string{orgGUID}},
	)

	if err != nil {
		return Space{}, Warnings(warnings), err
	}

	if len(spaces) == 0 {
		return Space{}, Warnings(warnings), actionerror.SpaceNotFoundError{Name: spaceName}
	}

	return actor.convertCCToActorSpace(spaces[0]), Warnings(warnings), nil
}

func (actor Actor) GetSpacesByGUIDs(guids ...string) ([]Space, Warnings, error) {

	currentV3Ver := actor.CloudControllerClient.CloudControllerAPIVersion()

	minSpacesGUIDsSupportVer, _ := semver.Make(ccversion.MinVersionSpacesGUIDsParamV3)

	guidsSupport := false
	queries := []ccv3.Query{}
	currentV3SemVer, err := semver.Make(currentV3Ver)
	if err == nil {
		guidsSupport = currentV3SemVer.GTE(minSpacesGUIDsSupportVer)
	}

	if guidsSupport {
		queries = []ccv3.Query{ccv3.Query{Key: ccv3.GUIDFilter, Values: guids}}
	}

	spaces, warnings, err := actor.CloudControllerClient.GetSpaces(queries...)

	var filteredSpaces []ccv3.Space
	guidToSpaces := map[string]ccv3.Space{}
	for _, space := range spaces {
		guidToSpaces[space.GUID] = space
	}

	for _, guid := range guids {
		filteredSpaces = append(filteredSpaces, guidToSpaces[guid])
	}
	spaces = filteredSpaces

	if err != nil {
		return []Space{}, Warnings(warnings), err
	}

	var v3Spaces []Space
	for _, ccSpace := range spaces {
		v3Spaces = append(v3Spaces, actor.convertCCToActorSpace(ccSpace))
	}

	return v3Spaces, Warnings(warnings), nil
}

func (actor Actor) convertCCToActorSpace(space ccv3.Space) Space {
	return Space{
		GUID:             space.GUID,
		Name:             space.Name,
		OrganizationGUID: space.Relationships[constant.RelationshipTypeOrganization].GUID,
	}
}

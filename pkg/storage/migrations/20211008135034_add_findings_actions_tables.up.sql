CREATE TABLE IF NOT EXISTS findings (
	id UUID PRIMARY KEY,
	identity_user varchar(1024) NOT NULL,
	identity_role varchar(1024) NOT NULL,
	identity_account varchar(50) NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL,
	event_count integer NOT NULL,
	status varchar(20) NOT NULL,
	document JSONB NOT NULL
);

CREATE TABLE IF NOT EXISTS events (
	id UUID PRIMARY KEY,
	time TIMESTAMPTZ NOT NULL,
	identity_user varchar(1024) NOT NULL,
	identity_role varchar(1024) NOT NULL,
	identity_account varchar(50) NOT NULL,
	data JSONB NOT NULL
);

CREATE TABLE IF NOT EXISTS actions (
	id UUID PRIMARY KEY,
	finding_id UUID NOT NULL REFERENCES findings,
	event_id UUID NOT NULL REFERENCES events,
	status varchar(20) NOT NULL,
	time TIMESTAMPTZ NOT NULL,
	has_recommendations BOOLEAN NOT NULL,
	enabled BOOLEAN NOT NULL
);

CREATE TABLE IF NOT EXISTS least_privilege_policies (
	id UUID PRIMARY KEY,
	action_id UUID NOT NULL REFERENCES actions,
	aws_policy JSONB NOT NULL,
	comment varchar(200) NOT NULL,
	role_name varchar(1024) NOT NULL
);

ALTER TABLE IF EXISTS actions ADD COLUMN selected_least_privilege_policy_id UUID REFERENCES least_privilege_policies;

CREATE TABLE IF NOT EXISTS cloud_resource_instances (
	id UUID PRIMARY KEY,
	name varchar(200) NOT NULL,
	arn varchar(1024) NOT NULL
);

CREATE TABLE IF NOT EXISTS cloudresourceinstances_leastprivilegepolicies (
	id UUID PRIMARY KEY,
	cloudresourceinstance_id UUID NOT NULL REFERENCES cloud_resource_instances,
	leastprivilegepolicy_id UUID NOT NULL REFERENCES least_privilege_policies
);
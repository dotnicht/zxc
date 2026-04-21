package zxc.authz

import rego.v1

default allow := false
default reveal_secret := false
default reason := "policy denied"

decision := {
	"allow": allow,
	"reveal_secret": reveal_secret,
	"reason": reason,
}

is_root if input.subject.is_root

is_system if input.subject.system

is_tenant_owner if {
	input.tenant.owner_id != ""
	input.subject.id == input.tenant.owner_id
}

is_resource_owner if {
	input.resource.owner_id != ""
	input.subject.id == input.resource.owner_id
}

owns_release_dependencies if {
	input.related.target_owner_id != ""
	input.related.payload_owner_id != ""
	input.subject.id == input.related.target_owner_id
	input.subject.id == input.related.payload_owner_id
}

is_authenticated_tenant_user if {
	not is_root
	not is_system
	input.subject.id != ""
	input.tenant.id != ""
}

allow if {
	input.action in {
		"tenant.create",
		"tenant.get",
		"tenant.update",
		"tenant.delete",
		"tenant.list",
		"tenant.search",
	}
	is_root
}

allow if {
	input.action in {
		"worker.create",
		"worker.get",
		"worker.update",
		"worker.delete",
		"worker.list",
		"worker.search",
		"worker.assign_tenant",
		"worker.unassign_tenant",
		"worker.list_tenants",
		"worker.list_workers_for_tenant",
	}
	is_root
}

allow if {
	input.action == "user.create"
	is_tenant_owner
}

allow if {
	input.action == "user.get"
	is_tenant_owner
}

allow if {
	input.action == "user.get"
	is_resource_owner
}

allow if {
	input.action == "user.update"
	is_tenant_owner
}

allow if {
	input.action == "user.update"
	is_resource_owner
}

allow if {
	input.action == "user.delete"
	is_tenant_owner
}

allow if {
	input.action in {"user.list", "user.search"}
	is_tenant_owner
}

allow if {
	input.action in {"account.get", "account.list", "account.search"}
	is_authenticated_tenant_user
}

allow if {
	input.action == "session.create"
	is_tenant_owner
}

allow if {
	input.action in {"session.get", "session.list", "session.search"}
	is_authenticated_tenant_user
}

allow if {
	input.action in {"session.update", "session.delete"}
	is_tenant_owner
}

allow if {
	input.action == "target.create"
	is_tenant_owner
}

allow if {
	input.action in {"target.get", "target.list", "target.search"}
	is_authenticated_tenant_user
}

allow if {
	input.action in {"target.update", "target.delete"}
	is_tenant_owner
}

allow if {
	input.action in {"target.update", "target.delete"}
	is_resource_owner
}

reveal_secret if {
	input.action in {"target.get", "target.list", "target.search"}
	is_tenant_owner
}

reveal_secret if {
	input.action in {"target.get", "target.list", "target.search"}
	is_resource_owner
}

allow if {
	input.action == "payload.create"
	is_authenticated_tenant_user
}

allow if {
	input.action in {"payload.get", "payload.list", "payload.search"}
	is_authenticated_tenant_user
}

allow if {
	input.action in {"payload.update", "payload.delete"}
	is_tenant_owner
}

allow if {
	input.action in {"payload.update", "payload.delete"}
	is_resource_owner
}

allow if {
	input.action == "release.create"
	is_tenant_owner
}

allow if {
	input.action == "release.create"
	owns_release_dependencies
}

allow if {
	input.action in {"release.get", "release.list", "release.search"}
	is_authenticated_tenant_user
}

allow if {
	input.action == "release.deploy"
	is_tenant_owner
	input.resource.status == "unknown"
}

allow if {
	input.action == "release.deploy"
	is_resource_owner
	input.resource.status == "unknown"
}

allow if {
	input.action == "release.transition"
	is_system
	transition_allowed
}

transition_allowed if {
	input.resource.status == "wait"
	input.resource.next_status == "deployed"
}

transition_allowed if {
	input.resource.status == "deployed"
	input.resource.next_status == "alive"
}

transition_allowed if {
	input.resource.status == "wait"
	input.resource.next_status == "dead"
}

transition_allowed if {
	input.resource.status == "deployed"
	input.resource.next_status == "dead"
}

transition_allowed if {
	input.resource.status == "alive"
	input.resource.next_status == "dead"
}

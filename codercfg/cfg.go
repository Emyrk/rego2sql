package codercfg

import "github.com/Emyrk/rego2sql"

func GroupACLMatcher(m rego2sql.VariableMatcher) rego2sql.VariableMatcher {
	return ACLGroupMatcher(m, []string{"input", "object", "acl_group_list"}, []string{"group_acl"})
}

func UserACLMatcher(m rego2sql.VariableMatcher) rego2sql.VariableMatcher {
	return ACLGroupMatcher(m, []string{"input", "object", "acl_user_list"}, []string{"user_acl"})
}

/*

NOTE: THIS IS JUST AN EXAMPLE APP FOR TESTING!
During normal usage this "app" directory is overwritten (or bind-mounted)
from the user's app.

*/


// The "root" entity is a special psuedo entity decalres the root
// calls.
// All queries start here.
export var root = {
	edges: {
		viewer() {
			return {
				type: 'member',
				query: `select * from member where id = $identity limit 1`
			}
		},
		// The 'raw' type (default if ommited) lets you create arbitary sql responses
		oneToTen() {
			return {
				query: `
					select array_agg(a) as numbers
					from generate_series(1,10) a
				`
			}
		},
		members() {
			return {
				type: 'array',
				of: 'member',
				query: `
					select * from member
				`
			}
		}
	}
}

export var member = {
	properties: {
		username:   {type: 'text'},
		password:   {type:'text'},
		is_su:      {type: 'boolean',   def: 'false' },
	},
	edges: {
		email_addresses() {
			return {
				type: 'array',
				of: 'email_address',
				query: `
					select * from email_address
					where email_address.member_id = $id`
				,
			}
		},
		orgs() {
			return {
				type: 'array',
				of: 'org',
				query: `
					select org.* from org_member
					left join org on org_member.org_id = org.id
					where org_member.member_id = $this.id
				`
			}
		},
	},
}

export var member_email = {
	properties: {
		member_id: {type: 'uuid', ref:'member'},
		address: { type: 'text', unique: true },
		is_confirmed: { type: 'boolean' },
	},
	edges: {
		member() {
			return {
				type: 'member',
				query: [`
					select * from member where id = $1
				`, this.member_id]
			}
		},
	},
};

export var org = {
	properties: {
		name: { type: 'text' },
		domain: { type: 'text', index:true, nullable: true, def: null },
		is_company: { type: 'boolean' },
	},
	edges: {
		members() {
			return {
				type: 'array',
				of: 'member',
				query: `
					select member.*
					from org_member
					left join member on member.id = org_member.member_id
					where org_member.org_id = $this.id
				`
			}
		},
	},
	indexes: {
		domain: {
			unique: true,
			on: [ 'domain' ]
		}
	},
};

export var org_member = {
	properties: {
		org_id: {type: 'uuid', ref:'org'},
		member_id: {type: 'uuid', ref:'member'},
		is_admin: { type: 'boolean' },
		is_confirmed: { type: 'boolean' },
	}
};

export var team = {
	properties: {
		name: { type: 'text' },
	},
};

export var team_member = {
	properties: {
		team_id: {type: 'uuid', ref:'team'},
		member_id: {type: 'uuid', ref:'member'},
		is_admin: { type: 'text' },
	}
};

export var client = {
	properties: {
		first_name: { type: 'text' },
		last_name: { type: 'text' },
	},
};

export var issue = {
	properties: {
		name: { type: 'text' },
	},
};

export var exercise = {
	properties: {
		name: { type: 'text' },
	},
};

export var client_case = {
	properties: {
		client_id: { type: 'uuid', ref: 'client' },
		org_id: { type: 'uuid', ref: 'org' },
		issue_id: { type: 'uuid', ref: 'issue' },
	}
};

export var case_member = {
	properties: {
		is_admin: { type: 'boolean' },
	}
};

export var case_team = {
	properties: {
		case_id: {type: 'uuid', ref: 'client_case'},
		team_id: {type: 'uuid', ref: 'team'}
	}
};

export var course = {
	properties: {
		case_id: { type: 'uuid', ref: 'client_case' },
		name: { type: 'text' },
	},
};

export var assignment = {
	properties: {
		course_id: { type: 'uuid', ref: 'course' },
		name: { type: 'text' },
		due_at: { type: 'timestamptz' },
	},
};

export var task = {
	properties: {
		assignment_id: { type: 'uuid', ref: 'assignment' },
		execise_id: { type: 'uuid', ref: 'exercise', nullable: true },
		name: { type: 'text' },
		is_complete: { type: 'boolean', nullable: true, def: null },
		instruments: { type: 'json' },
	},
};

export var data = {
	properties: {
		created_at: {type: 'timestamptz', def:'now()'},
		task_id: { type: 'uuid', ref: 'task' },
		value: { type: 'json' },
	},
	indexes: {
		created_at: {
			on: [ 'created_at' ]
		}
	}
};

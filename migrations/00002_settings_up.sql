-- This is where we init some configs about this node

-- First settings, this is generated from SureSQL SaaS
INSERT INTO _configs (id, label,ip, host, port, ssl, dbms, mode, nodes, node_number, is_init_done, is_split_write, encryption_method) 
 VALUES (1,'Test Project','127.0.0.1','medatech-dbone-master.happyrich.uk','',true,'RQLITE', 'rw', 1, 1, true, false, "none")
  ON CONFLICT(id) DO UPDATE SET label=excluded.label, mode=excluded.mode,
    nodes=excluded.nodes, node_number=excluded.node_number, 
    is_init_done=excluded.is_init_done, is_split_write=excluded.is_split_write,
    encryption_method=excluded.encryption_method;

-- Information about the peers in the format of
-- node_number|hostname|ip|mode   and the CONFIG_NODE_DELIMITER in this case is "|"
-- If there is no cluster, still put 1 entry in here, for it's own node.
-- node_number == 0 is ALWAYS the master!
DELETE FROM _settings WHERE category='nodes';
INSERT INTO _settings(category, data_type, setting_key, text_value) VALUES 
('nodes', 'string', 'master', '0|medatech-dbone-master.happyrich.uk|127.0.0.1|rw'), 
('nodes', 'string', 'peer-01', '1|medatech-dbone-peer-01.happyrich.uk|127.0.0.1|r'), 
('nodes', 'string', 'peer-02', '2|medatech-dbone-peer-02.happyrich.uk|127.0.0.1|r');
INSERT INTO _settings(category, data_type, setting_key, text_value) VALUES 
('system', 'string', 'label', 'Test Project'),
('system', 'string', 'host', 'medatech-dbone-master.happyrich.uk'),
('system', 'string', 'ip', '127.0.0.1'),
('system', 'string', 'port', ''),
('system', 'string', 'dbms', 'RQLITE'),
('system', 'string', 'mode', 'rw'),
('system', 'string', 'encryption_method', "none");
INSERT INTO _settings(category, data_type, setting_key, int_value) VALUES
('system', 'bool', 'ssl', 1),
('system', 'bool', 'nodes', 1),
('system', 'bool', 'node_number', 1),
('system', 'bool', 'is_init_done', 1),
('system', 'bool', 'is_split_write', 0);
	

-- Information about user that can be used by the client's project
-- INSERT INTO _users (username, password, role_name) VALUES ()

INSERT INTO _settings(category, data_type, setting_key, int_value) VALUES ("connection", "int", "pool_on", true);
INSERT INTO _settings(category, data_type, setting_key, int_value) VALUES ("connection", "int", "max_pool", 25);
INSERT INTO _settings(category, data_type, setting_key, int_value) VALUES ("token", "int", "token_exp", 360); -- 6 hours
INSERT INTO _settings(category, data_type, setting_key, int_value) VALUES ("token", "int", "refresh_exp", 1440); -- 2 days
INSERT INTO _settings(category, data_type, setting_key, int_value) VALUES ("token", "int", "token_ttl", 5); -- 5 minutes

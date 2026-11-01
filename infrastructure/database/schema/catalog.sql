SELECT jsonb_build_object(
'columns', (SELECT jsonb_agg(jsonb_build_array(table_name,column_name,ordinal_position,data_type,udt_name,is_nullable,column_default,character_maximum_length,datetime_precision) ORDER BY table_name,ordinal_position) FROM information_schema.columns WHERE table_schema='public' AND table_name = ANY($1)),
'constraints', (SELECT jsonb_agg(jsonb_build_array(c.relname,k.conname,pg_get_constraintdef(k.oid),k.convalidated) ORDER BY c.relname,k.conname) FROM pg_constraint k JOIN pg_class c ON c.oid=k.conrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname = ANY($1)),
'indexes', (SELECT jsonb_agg(jsonb_build_array(tablename,indexname,indexdef) ORDER BY tablename,indexname) FROM pg_indexes WHERE schemaname='public' AND tablename = ANY($1))
)::text

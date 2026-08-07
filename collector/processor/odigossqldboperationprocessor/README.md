# SQL DB Operation Processor

The SQL DB Operation Processor is responsible for detecting the type of SQL operation performed by analyzing the SQL query within a trace. It extracts the query text, checks for common SQL operations (SELECT, INSERT, UPDATE, DELETE, CREATE), and assigns the appropriate operation name to the trace span.

This processor ensures that each trace contains clear metadata about the database operation performed, enabling better observability and trace analysis.

create table ishocon1.top_comments (
  id int NOT NULL AUTO_INCREMENT PRIMARY KEY,
  product_id int NOT NULL,
  user_name varchar(128) NOT NULL,
  content varchar(128) NOT NULL,
  created_at datetime NOT NULl
)

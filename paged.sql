create table ishocon1.paged_comments(
  id int NOT NULL AUTO_INCREMENT,
  page int NOT NULL,
  `product_id` int(11) NOT NULL,
  `user_id` int(11) NOT NULL,
  `content` varchar(128) NOT NULL,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


create table ishocon1.paged_products(
  `id` int(11) NOT NULL AUTO_INCREMENT,
  page int(11) NOT NULL,
  `name` varchar(64) NOT NULL,
  `description` text,
  `image_path` varchar(32) NOT NULL,
  `price` int(8) NOT NULL,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


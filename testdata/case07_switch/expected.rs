match status {
    Status::Active              => println!("Active"),
    Status::Inactive            => println!("Inactive"),
    Status::PendingVerification => println!("Pending"),
    Status::Suspended           => println!("Suspended"),
}

package interfaces

type PasswordChecker interface {
	CheckPassword(hashedPassword, plainPassword string) bool
}

type PasswordHasher interface {
	HashPassword(plainPassword string) (string, error)
}

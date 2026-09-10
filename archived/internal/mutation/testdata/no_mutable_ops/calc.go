package nomut

// Mask uses a compound-assignment operator `&^=`, which Go represents as a single
// token.AND_NOT_ASSIGN inside an *ast.AssignStmt — NOT an *ast.BinaryExpr. The
// oracle only mutates binary/unary-NOT expressions, so this line has zero mutable
// sites (and stays that way no matter which binary operators become supported).
// The oracle must report that as "no signal", not as "all mutants killed" (which
// would falsely imply the line is tested).
func Mask(x uint) uint {
	x &^= 2
	return x
}

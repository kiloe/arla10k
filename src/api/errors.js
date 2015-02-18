
var err = {};

err.InvalidUserId = 'invalid user id';
err.InvalidPassword = 'invalid password';
err.InvalidToken = 'invalid token';
err.TokenExpired = 'token expired';

err.PasswordTooShort = 'passwords must be at least 9 characters';
err.PasswordTooNumeric = 'passwords should not be purely numeric';
err.PasswordTooSimple = 'passwords under 16 characters must mix case, include numbers or use more unusual characters';

err.LockedUserId = 'user id has been temporarily locked due to too many failed logins';

module.exports = err;

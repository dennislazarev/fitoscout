package auth

// closedModules — замкнутые модули (MASTER.md, раздел 10):
// полный CRUD доступен Android-клиенту, так как это его рабочие данные.
// Справочные модули (растения, болезни, ...) Android может только читать.
var closedModules = map[string]bool{
        "registry": true,
        "calendar": true,
        "library":  true,
        "comments": true,
}

// IsClosedModule сообщает, является ли модуль замкнутым.
func IsClosedModule(moduleKey string) bool {
        return closedModules[moduleKey]
}

// CanRead сообщает, может ли роль читать модуль.
// Чтение доступно всем распознаваемым ролям.
func CanRead(role Role, _ string) bool {
        return role == RoleWeb || role == RoleAndroid
}

// CanWrite сообщает, может ли роль писать в модуль.
// web — всюду; android — только в замкнутые модули.
func CanWrite(role Role, moduleKey string) bool {
        switch role {
        case RoleWeb:
                return true
        case RoleAndroid:
                return IsClosedModule(moduleKey)
        default:
                return false
        }
}
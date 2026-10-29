// Package topology предоставляет примитивы и функции для сбора топологии базы данных.
package topology

// StaticSeedProvider возвращает статический список начальных адресов хостов баз данных.
func StaticSeedProvider() []string {
	pgHosts := make([]string, 0, 1)

	return pgHosts
}

// GetTopology выполняет опрос и сбор текущей топологии базы данных.
func GetTopology() {

}

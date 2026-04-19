const piniaPersistConfig = (key: string) => {
	const persist = {
		key,
		storage: window.localStorage
	}
	return persist
}

export default piniaPersistConfig

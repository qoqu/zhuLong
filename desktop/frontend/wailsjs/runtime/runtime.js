// @ts-check
/**
 * Call a Go method
 * @param {string} method
 * @param {...any} args
 * @returns {Promise<any>}
 */
export function Call(method, ...args) {
    return window['go'][method](...args);
}

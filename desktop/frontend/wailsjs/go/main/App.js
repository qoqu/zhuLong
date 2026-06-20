// @ts-check
import {Call} from '../runtime/runtime.js';

/**
 * Greet
 * @param {string} arg1
 * @returns {Promise<string>}
 */
export function Greet(arg1) {
    return Call('main.App.Greet', arg1);
}

/**
 * GetVersion
 * @returns {Promise<string>}
 */
export function GetVersion() {
    return Call('main.App.GetVersion');
}

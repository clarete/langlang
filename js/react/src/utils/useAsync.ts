import { AnyActionArg, useCallback, useEffect, useReducer, useRef } from 'react'

enum ActionTypes {
    PENDING = "PENDING",
    SUCCESS = "SUCCESS",
    ERROR = "ERROR",
}

export interface AsyncState<T> {
    status: "idle" | "pending" | "success" | "error";
    data: T | null;
    error: Error | null;
}

interface PendingAction {
    type: ActionTypes.PENDING;
}

interface SuccessAction<T> {
    type: ActionTypes.SUCCESS;
    payload: T;
}

interface ErrorAction {
    type: ActionTypes.ERROR;
    payload: Error;
}

type AsyncAction<T> = PendingAction | SuccessAction<T> | ErrorAction;

function asyncReducer<T>(state: AsyncState<T>, action: AsyncAction<T>): AsyncState<T> {
    switch (action.type) {
        case ActionTypes.PENDING:
            return { ...state, status: "pending" };
        case ActionTypes.SUCCESS:
            return { ...state, status: "success", data: action.payload };
        case ActionTypes.ERROR:
            return { ...state, status: "error", error: action.payload };
        default:
            return state;
    }
}

export function useAsync<T>(promise: () => Promise<T>, autoExecute = true) {
    const [state, dispatch] = useReducer(asyncReducer, {
        status: "idle",
        data: null,
        error: null,
    } as AsyncState<T>);

    const isMounted = useRef(true)
    const promiseRef = useRef(promise);
    const pendingActionRef = useRef<AsyncAction<T> | null>(null);


    useEffect(() => {
        promiseRef.current = promise;
    }, [promise]);

    useEffect(() => {
        isMounted.current = true;
        return () => {
            isMounted.current = false;
        }
    }, []);


    const execute = useCallback(async () => {
        dispatch({ type: ActionTypes.PENDING });


        try {
            const data = await promiseRef.current();

            if (isMounted.current) {
                dispatch({ type: ActionTypes.SUCCESS, payload: data });
            } else {
                pendingActionRef.current = { type: ActionTypes.SUCCESS, payload: data };
            }

            return data;
        } catch (error) {
            if (isMounted.current) {
                dispatch({ type: ActionTypes.ERROR, payload: error as Error });
            } else {
                pendingActionRef.current = { type: ActionTypes.ERROR, payload: error as Error };
            }
            throw error;
        }
    }, []);

    useEffect(() => {
        if (autoExecute) {
            execute();
        }

    }, [autoExecute, execute]);

    return {
        execute,
        status: state.status,
        data: state.data,
        error: state.error,
        isIdle: state.status === "idle",
        isPending: state.status === "pending",
        isSuccess: state.status === "success",
        isError: state.status === "error",
    };
}

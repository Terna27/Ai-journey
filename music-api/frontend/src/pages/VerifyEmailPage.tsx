import {
    useEffect,
    useRef,
    useState,
} from 'react'

import {
    Link,
    useSearchParams,
} from 'react-router-dom'

import { verifyEmail } from '../lib/api'

type VerificationState =
    | 'verifying'
    | 'success'
    | 'error'

function VerifyEmailPage() {
    const [searchParams] =
        useSearchParams()

    const [status, setStatus] =
        useState<VerificationState>(
            'verifying',
        )

    const [message, setMessage] =
        useState(
            'Verifying your email address...',
        )

    const hasStarted =
        useRef(false)

    useEffect(() => {
        if (hasStarted.current) {
            return
        }

        hasStarted.current = true

        const tokenParam =
            searchParams.get('token')

        if (!tokenParam) {
            setStatus('error')
            setMessage(
                'This verification link is invalid.',
            )
            return
        }

        // Create a definite string value here.
        // This prevents TypeScript from treating
        // the token as string | null inside the
        // asynchronous function below.
        const token: string = tokenParam

        async function runVerification() {
            try {
                const result =
                    await verifyEmail({
                        token,
                    })

                setStatus('success')
                setMessage(
                    result.message,
                )
            } catch (err) {
                setStatus('error')

                setMessage(
                    err instanceof Error
                        ? err.message
                        : 'Unable to verify your email address.',
                )
            }
        }

        void runVerification()
    }, [searchParams])

    return (
        <section className="form-page">
            <div className="form-card">
                <p className="eyebrow">
                    EMAIL VERIFICATION
                </p>

                {status === 'verifying' && (
                    <>
                        <h2>
                            Verifying email
                        </h2>

                        <p className="form-description">
                            {message}
                        </p>
                    </>
                )}

                {status === 'success' && (
                    <>
                        <h2>
                            Email verified
                        </h2>

                        <div
                            className="status-message success-message"
                            role="status"
                        >
                            {message}
                        </div>

                        <p className="form-description">
                            Your email address has been
                            verified. You can now sign in
                            to your account.
                        </p>

                        <Link
                            to="/login"
                            className="primary-button"
                        >
                            Sign in
                        </Link>
                    </>
                )}

                {status === 'error' && (
                    <>
                        <h2>
                            Verification failed
                        </h2>

                        <div
                            className="form-error"
                            role="alert"
                        >
                            {message}
                        </div>

                        <p className="form-description">
                            The verification link may be
                            invalid or expired. Sign in
                            to your account to request
                            another verification email.
                        </p>

                        <Link
                            to="/login"
                            className="primary-button"
                        >
                            Go to sign in
                        </Link>
                    </>
                )}
            </div>
        </section>
    )
}

export default VerifyEmailPage